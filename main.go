// Command hypedoc renders HTML graphics into screenshots and pins them to the
// top of a markdown document, preserving everything else in it.
package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/stuyk/hypedoc/internal/assets"
	"github.com/stuyk/hypedoc/internal/doc"
	"github.com/stuyk/hypedoc/internal/render"
)

// version.txt is the single source of truth; bumping it on main cuts a release.
//
//go:embed version.txt
var versionFile string

var version = strings.TrimSpace(versionFile)

const usage = `HypeDoc — turn HTML into high-fidelity README graphics.

Usage:
  hypedoc build [flags] [file.html ...]   render graphics and update the doc
  hypedoc init  [--doc README.md]         create a starter .hypedoc/01-hero.html
  hypedoc strip [--doc README.md]         remove the generated block from the doc
  hypedoc guide                           print the authoring guide (for AI agents)
  hypedoc version

Run "hypedoc <command> -h" for flags.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	var err error
	switch cmd, args := os.Args[1], os.Args[2:]; cmd {
	case "build":
		err = runBuild(args)
	case "init":
		err = runInit(args)
	case "strip":
		err = runStrip(args)
	case "guide":
		fmt.Print(assets.Guide)
	case "version", "--version", "-v":
		fmt.Println("hypedoc", version)
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", cmd, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "hypedoc:", err)
		os.Exit(1)
	}
}

type graphic struct {
	Source string `json:"source"`
	Alt    string `json:"alt"`
	Light  string `json:"light"`
	Dark   string `json:"dark,omitempty"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type buildReport struct {
	Doc      string    `json:"doc"`
	Graphics []graphic `json:"graphics"`
	Removed  []string  `json:"removed,omitempty"`
	Changed  bool      `json:"changed"`
}

func runBuild(args []string) error {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	docPath := fs.String("doc", "README.md", "markdown document to update")
	srcDir := fs.String("src", "", "directory of .html sources (default <docdir>/.hypedoc)")
	outDir := fs.String("out", "", "directory for rendered PNGs (default <src>/img)")
	width := fs.Int("width", render.DefaultWidth, "default CSS width when a file has no hypedoc:width meta")
	scale := fs.Float64("scale", 2, "device pixel ratio")
	theme := fs.String("theme", "auto", "auto|light|dark|both (auto = both if the HTML uses prefers-color-scheme)")
	opaque := fs.Bool("opaque", false, "render on a white page background instead of transparent")
	browser := fs.String("browser", "", "Chrome/Chromium executable (default: auto-detect, else download)")
	timeout := fs.Duration("timeout", 30*time.Second, "per-screenshot timeout")
	dryRun := fs.Bool("dry-run", false, "render images but print the updated doc instead of writing it")
	asJSON := fs.Bool("json", false, "print a JSON report")
	fs.Parse(args)

	switch *theme {
	case "auto", "light", "dark", "both":
	default:
		return fmt.Errorf("invalid --theme %q", *theme)
	}

	docDir := filepath.Dir(*docPath)
	if *srcDir == "" {
		*srcDir = filepath.Join(docDir, ".hypedoc")
	}
	if *outDir == "" {
		*outDir = filepath.Join(*srcDir, "img")
	}

	sources := fs.Args()
	explicit := len(sources) > 0
	if !explicit {
		var err error
		if sources, err = filepath.Glob(filepath.Join(*srcDir, "*.html")); err != nil {
			return err
		}
		sort.Strings(sources)
	}
	if len(sources) == 0 {
		return fmt.Errorf("no .html files in %s (run `hypedoc init` or pass files)", *srcDir)
	}

	original, err := os.ReadFile(*docPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	logf := func(f string, a ...any) {
		if !*asJSON {
			fmt.Fprintf(os.Stderr, f+"\n", a...)
		}
	}
	r, err := render.New(render.Options{
		Browser: *browser, Scale: *scale, Width: *width, Transparent: !*opaque, Timeout: *timeout, Logf: logf,
	})
	if err != nil {
		return err
	}
	defer r.Close()

	report := buildReport{Doc: *docPath}
	var images []doc.Image
	produced := map[string]bool{}
	for _, src := range sources {
		html, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		schemes := schemesFor(*theme, string(html))
		base := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
		g := graphic{Source: src}
		for _, s := range schemes {
			out := filepath.Join(*outDir, base+".png")
			if len(schemes) > 1 {
				out = filepath.Join(*outDir, base+"."+s+".png")
			}
			res, err := r.Capture(render.Job{Src: src, Out: out, Scheme: s})
			if err != nil {
				return err
			}
			produced[filepath.Clean(out)] = true
			rel, err := filepath.Rel(docDir, out)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if s == "dark" && len(schemes) > 1 {
				g.Dark = rel
			} else {
				g.Light = rel
			}
			g.Alt, g.Width, g.Height = res.Alt, res.Width, res.Height
			logf("rendered %s (%s, %dx%d)", out, s, res.Width, res.Height)
		}
		if g.Alt == "" {
			g.Alt = base
		}
		report.Graphics = append(report.Graphics, g)
		images = append(images, doc.Image{Alt: g.Alt, Light: g.Light, Dark: g.Dark, Width: g.Width})
	}

	// Remove PNGs left over from deleted or renamed sources (full builds only).
	if !explicit {
		old, _ := filepath.Glob(filepath.Join(*outDir, "*.png"))
		for _, p := range old {
			if !produced[filepath.Clean(p)] {
				if err := os.Remove(p); err == nil {
					report.Removed = append(report.Removed, p)
				}
			}
		}
	}

	updated, err := doc.Inject(string(original), doc.RenderBlock(images))
	if err != nil {
		return fmt.Errorf("%s: %w", *docPath, err)
	}
	report.Changed = updated != string(original)

	switch {
	case *dryRun:
		fmt.Print(updated)
	case report.Changed:
		if err := os.WriteFile(*docPath, []byte(updated), 0o644); err != nil {
			return err
		}
	}
	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}
	if !*dryRun {
		logf("updated %s (%d graphics)", *docPath, len(images))
	}
	return nil
}

func schemesFor(theme, html string) []string {
	switch theme {
	case "light", "dark":
		return []string{theme}
	case "both":
		return []string{"light", "dark"}
	}
	if themedRe.MatchString(html) {
		return []string{"light", "dark"}
	}
	return []string{"light"}
}

// themedRe matches real dark-mode hooks (CSS media queries or matchMedia calls),
// not prose that merely mentions prefers-color-scheme.
var themedRe = regexp.MustCompile(`(@media[^{]*|matchMedia\(\s*['"][^'"]*|<source[^>]*media=["'][^"']*)\(\s*prefers-color-scheme`)

var headingRe = regexp.MustCompile(`(?m)^#\s+(.+?)\s*$`)

func runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	docPath := fs.String("doc", "README.md", "markdown document the graphics are for")
	force := fs.Bool("force", false, "overwrite an existing starter file")
	fs.Parse(args)

	docDir := filepath.Dir(*docPath)
	title := ""
	if b, err := os.ReadFile(*docPath); err == nil {
		if m := headingRe.FindSubmatch(b); m != nil {
			title = string(m[1])
		}
	}
	if title == "" {
		abs, _ := filepath.Abs(docDir)
		title = filepath.Base(abs)
	}

	dest := filepath.Join(docDir, ".hypedoc", "01-hero.html")
	if _, err := os.Stat(dest); err == nil && !*force {
		return fmt.Errorf("%s already exists (use --force)", dest)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	content := strings.ReplaceAll(assets.Starter, "{{TITLE}}", htmlEscape(title))
	if err := os.WriteFile(dest, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("created %s\nnext: edit it (see `hypedoc guide`), then run `hypedoc build --doc %s`\n", dest, *docPath)
	return nil
}

func runStrip(args []string) error {
	fs := flag.NewFlagSet("strip", flag.ExitOnError)
	docPath := fs.String("doc", "README.md", "markdown document to clean")
	fs.Parse(args)

	b, err := os.ReadFile(*docPath)
	if err != nil {
		return err
	}
	out, found, err := doc.Strip(string(b))
	if err != nil {
		return err
	}
	if !found {
		fmt.Println("no hypedoc block in", *docPath)
		return nil
	}
	if err := os.WriteFile(*docPath, []byte(out), 0o644); err != nil {
		return err
	}
	fmt.Println("removed hypedoc block from", *docPath)
	return nil
}

func htmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(s)
}
