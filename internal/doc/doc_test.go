package doc

import (
	"strings"
	"testing"
)

func TestInjectPrependsAndPreserves(t *testing.T) {
	src := "# Title\n\nBody text.\n"
	block := RenderBlock([]Image{{Alt: "Hero", Light: ".hypedoc/img/hero.png", Width: 960}})
	out, err := Inject(src, block)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, StartMarker) {
		t.Fatalf("block not at top:\n%s", out)
	}
	if !strings.HasSuffix(out, "\n\n"+src) {
		t.Fatalf("original content not preserved:\n%s", out)
	}
}

func TestInjectIsIdempotent(t *testing.T) {
	src := "# Title\n"
	block := RenderBlock([]Image{{Alt: "A", Light: "a.png", Width: 100}})
	once, _ := Inject(src, block)
	twice, _ := Inject(once, block)
	if once != twice {
		t.Fatalf("not idempotent:\n%s\n---\n%s", once, twice)
	}
}

func TestInjectReplacesExistingBlock(t *testing.T) {
	src := "intro\n" + StartMarker + "\nold\n" + EndMarker + "\n\n# Title\n"
	out, err := Inject(src, RenderBlock(nil))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "old") || !strings.HasPrefix(out, "intro\n") || !strings.HasSuffix(out, "\n\n# Title\n") {
		t.Fatalf("bad replace:\n%s", out)
	}
}

func TestInjectAfterFrontMatter(t *testing.T) {
	src := "---\ntitle: x\n---\n# Title\n"
	out, _ := Inject(src, RenderBlock(nil))
	if !strings.HasPrefix(out, "---\ntitle: x\n---\n\n"+StartMarker) {
		t.Fatalf("front matter not respected:\n%s", out)
	}
}

func TestInjectUnterminated(t *testing.T) {
	if _, err := Inject(StartMarker+"\nno end", "x"); err == nil {
		t.Fatal("expected error")
	}
}

func TestInjectIgnoresMarkersInProseAndCode(t *testing.T) {
	src := "# Title\n\nBlock lives between `" + StartMarker + "` and `" + EndMarker + "`.\n\n```html\n" +
		StartMarker + "\n" + EndMarker + "\n```\n"
	out, err := Inject(src, RenderBlock(nil))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, StartMarker) || !strings.HasSuffix(out, "\n\n"+src) {
		t.Fatalf("markers in prose/code were treated as the block:\n%s", out)
	}
	again, _ := Inject(out, RenderBlock(nil))
	if again != out {
		t.Fatalf("not idempotent with prose markers:\n%s", again)
	}
}

func TestStripRoundTrip(t *testing.T) {
	src := "# Title\n\nBody\n"
	withBlock, _ := Inject(src, RenderBlock([]Image{{Alt: "A", Light: "a.png", Dark: "a.dark.png", Width: 1}}))
	out, ok, err := Strip(withBlock)
	if err != nil || !ok || out != src {
		t.Fatalf("strip mismatch (ok=%v err=%v):\n%q", ok, err, out)
	}
}

func TestRenderBlockEscapes(t *testing.T) {
	b := RenderBlock([]Image{{Alt: `a "quoted" <tag>`, Light: "dir with space/x.png", Width: 1}})
	if !strings.Contains(b, `src="dir%20with%20space/x.png"`) || !strings.Contains(b, "&#34;quoted&#34; &lt;tag&gt;") {
		t.Fatalf("not escaped:\n%s", b)
	}
}
