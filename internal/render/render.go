// Package render turns self-contained HTML files into high-DPI PNG screenshots
// using a headless Chromium driven over the DevTools protocol.
package render

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

const (
	DefaultWidth = 960
	maxHeight    = 16384
)

type Options struct {
	Browser     string  // explicit Chromium/Chrome path; empty = auto-detect, then download
	Scale       float64 // device pixel ratio
	Width       int     // default CSS viewport width when a file has no hypedoc:width meta
	Transparent bool    // transparent page background (lets rounded cards float)
	Timeout     time.Duration
	Logf        func(format string, args ...any)
}

// Job is one screenshot of one HTML file in one color scheme.
type Job struct {
	Src    string // HTML file path
	Out    string // PNG output path
	Scheme string // "light" or "dark"
}

// Result describes what was captured.
type Result struct {
	Alt    string
	Width  int // CSS pixels
	Height int // CSS pixels
}

type Renderer struct {
	opts     Options
	launcher *launcher.Launcher
	browser  *rod.Browser
}

func New(opts Options) (*Renderer, error) {
	if opts.Scale <= 0 {
		opts.Scale = 2
	}
	if opts.Width <= 0 {
		opts.Width = DefaultWidth
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}
	if opts.Logf == nil {
		opts.Logf = func(string, ...any) {}
	}

	l := launcher.New().Headless(true).NoSandbox(os.Geteuid() == 0)
	bin := opts.Browser
	if bin == "" {
		bin = os.Getenv("HYPEDOC_BROWSER")
	}
	if bin == "" {
		if p, ok := launcher.LookPath(); ok {
			bin = p
		}
	}
	if bin == "" {
		if cached := launcher.NewBrowser(); cached.Validate() == nil {
			bin = cached.BinPath()
		}
	}
	if bin != "" {
		l = l.Bin(bin)
	} else {
		opts.Logf("no Chrome/Chromium found; downloading a pinned Chromium (one time)...")
	}

	u, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("launch browser: %w (set --browser or HYPEDOC_BROWSER)", err)
	}
	b := rod.New().ControlURL(u)
	if err := b.Connect(); err != nil {
		l.Kill()
		return nil, fmt.Errorf("connect browser: %w", err)
	}
	return &Renderer{opts: opts, launcher: l, browser: b}, nil
}

func (r *Renderer) Close() {
	_ = r.browser.Close()
	r.launcher.Kill()
}

type meta struct {
	Width    string `json:"width"`
	Selector string `json:"selector"`
	Alt      string `json:"alt"`
	Wait     string `json:"wait"`
}

type rect struct {
	X, Y, W, H, DocH float64
}

func (r *Renderer) Capture(job Job) (Result, error) {
	abs, err := filepath.Abs(job.Src)
	if err != nil {
		return Result{}, err
	}

	page, err := r.browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return Result{}, err
	}
	defer page.Close()
	page = page.Timeout(r.opts.Timeout)

	if r.opts.Transparent {
		zero := 0.0
		if err := (proto.EmulationSetDefaultBackgroundColorOverride{Color: &proto.DOMRGBA{A: &zero}}).Call(page); err != nil {
			return Result{}, err
		}
	}
	scheme := job.Scheme
	if scheme == "" {
		scheme = "light"
	}
	if err := (proto.EmulationSetEmulatedMedia{Features: []*proto.EmulationMediaFeature{
		{Name: "prefers-color-scheme", Value: scheme},
		{Name: "prefers-reduced-motion", Value: "reduce"},
	}}).Call(page); err != nil {
		return Result{}, err
	}
	if err := r.viewport(page, r.opts.Width, 800); err != nil {
		return Result{}, err
	}

	fileURL := (&url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}).String()
	if err := page.Navigate(fileURL); err != nil {
		return Result{}, fmt.Errorf("open %s: %w", job.Src, err)
	}
	if err := page.WaitLoad(); err != nil {
		return Result{}, err
	}

	var m meta
	res, err := page.Eval(`() => {
		const m = n => (document.querySelector('meta[name="hypedoc:' + n + '"]') || {}).content || "";
		return { width: m("width"), selector: m("selector"), alt: m("alt") || document.title, wait: m("wait") };
	}`)
	if err != nil {
		return Result{}, err
	}
	if err := json.Unmarshal([]byte(res.Value.JSON("", "")), &m); err != nil {
		return Result{}, err
	}

	width := r.opts.Width
	if m.Width != "" {
		if _, err := fmt.Sscanf(m.Width, "%d", &width); err != nil || width <= 0 {
			return Result{}, fmt.Errorf("%s: invalid hypedoc:width %q", job.Src, m.Width)
		}
	}
	selector := m.Selector
	if selector == "" {
		selector = "body"
	}

	// Lay out at the target width, then grow the viewport to the full document
	// height so nothing is clipped and no scrolling occurs during capture.
	if err := r.viewport(page, width, 800); err != nil {
		return Result{}, err
	}
	if err := settle(page, m.Wait); err != nil {
		return Result{}, err
	}
	box, err := measure(page, selector)
	if err != nil {
		return Result{}, fmt.Errorf("%s: %w", job.Src, err)
	}
	if err := r.viewport(page, width, int(math.Min(math.Ceil(box.DocH), maxHeight))); err != nil {
		return Result{}, err
	}
	if err := settle(page, ""); err != nil {
		return Result{}, err
	}
	if box, err = measure(page, selector); err != nil {
		return Result{}, fmt.Errorf("%s: %w", job.Src, err)
	}
	if box.W < 1 || box.H < 1 {
		return Result{}, fmt.Errorf("%s: selector %q has empty size", job.Src, selector)
	}

	png, err := page.Screenshot(false, &proto.PageCaptureScreenshot{
		Format:                proto.PageCaptureScreenshotFormatPng,
		Clip:                  &proto.PageViewport{X: box.X, Y: box.Y, Width: box.W, Height: box.H, Scale: 1},
		CaptureBeyondViewport: true,
	})
	if err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(filepath.Dir(job.Out), 0o755); err != nil {
		return Result{}, err
	}
	if err := os.WriteFile(job.Out, png, 0o644); err != nil {
		return Result{}, err
	}
	return Result{Alt: m.Alt, Width: int(math.Round(box.W)), Height: int(math.Round(box.H))}, nil
}

func (r *Renderer) viewport(page *rod.Page, w, h int) error {
	return proto.EmulationSetDeviceMetricsOverride{Width: w, Height: h, DeviceScaleFactor: r.opts.Scale}.Call(page)
}

// settle waits for web fonts, an optional author-specified delay, and two frames.
func settle(page *rod.Page, waitMS string) error {
	_, err := page.Eval(`(ms) => document.fonts.ready
		.then(() => new Promise(r => setTimeout(r, parseInt(ms, 10) || 0)))
		.then(() => new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r))))`, waitMS)
	return err
}

func measure(page *rod.Page, selector string) (rect, error) {
	res, err := page.Eval(`(sel) => {
		const el = document.querySelector(sel);
		if (!el) return null;
		const b = el.getBoundingClientRect();
		const d = document.documentElement;
		return { X: b.left + scrollX, Y: b.top + scrollY, W: b.width, H: b.height,
		         DocH: Math.max(d.scrollHeight, document.body.scrollHeight, b.bottom + scrollY) };
	}`, selector)
	if err != nil {
		return rect{}, err
	}
	raw := res.Value.JSON("", "")
	if raw == "null" {
		return rect{}, fmt.Errorf("selector %q matched nothing", selector)
	}
	var out rect
	return out, json.Unmarshal([]byte(raw), &out)
}
