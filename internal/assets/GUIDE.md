# HypeDoc authoring guide (for AI agents)

HypeDoc renders HTML files into crisp PNG graphics and pins them to the top of a
markdown document inside a managed block. Everything below the block is never touched.

## Workflow

1. `hypedoc init --doc README.md` — creates `.hypedoc/01-hero.html` next to the doc.
2. Edit or add HTML files in `.hypedoc/`. They render in filename order
   (`01-hero.html`, `02-features.html`, ...).
3. `hypedoc build --doc README.md` — renders PNGs into `.hypedoc/img/` and
   writes/updates the block between `<!-- hypedoc:start -->` and `<!-- hypedoc:end -->`.
4. Commit the `.html` sources, the PNGs, and the updated doc.

Use `--json` for machine-readable output and `--dry-run` to print the doc without writing it.
`hypedoc strip` removes the block again.

## Rules for great graphics

- **Self-contained.** Inline all CSS in a `<style>` tag. Inline SVG for icons/diagrams.
  Avoid network fonts and remote images; renders must be deterministic and offline-safe.
  Use a system font stack: `ui-sans-serif, system-ui, -apple-system, "Segoe UI", Inter, Roboto, sans-serif`.
- **Fixed width.** Design for one CSS width (default 960px; GitHub's readme column is ~880px).
  Override per file with `<meta name="hypedoc:width" content="1200">`.
  Height is automatic — never use `vh` units or `height: 100%` on `body`.
- **Readable when scaled.** Body text >= 16px, headings >= 28px. The image may display smaller than designed.
- **Floating cards.** The page background is transparent by default. Give `body` `margin: 0`
  and a padded wrapper; put backgrounds/rounded corners on the card itself so it sits cleanly
  on both GitHub light and dark themes.
- **Light + dark.** If the file contains `prefers-color-scheme`, HypeDoc renders both themes
  and emits a `<picture>` that follows the reader's GitHub theme. Use CSS variables:
  ```css
  :root { --bg: #fff; --fg: #0b0d12; }
  @media (prefers-color-scheme: dark) { :root { --bg: #0d1117; --fg: #e6edf3; } }
  ```
- **No motion.** Animations are captured mid-flight. `prefers-reduced-motion: reduce` is emulated;
  respect it, or show the final state directly.
- **One idea per file.** Hero, feature grid, architecture diagram, CLI demo, comparison table —
  each as its own file. 2–4 graphics is the sweet spot.
- **Real content.** Pull names, commands, and claims from the project itself. No lorem ipsum.

## Per-file meta tags

| Meta | Default | Effect |
|---|---|---|
| `hypedoc:width` | `960` | CSS viewport width |
| `hypedoc:selector` | `body` | Element to capture (e.g. `#card`) |
| `hypedoc:alt` | `<title>` | Image alt text |
| `hypedoc:wait` | `0` | Extra milliseconds to wait before capture |

## Flags (build)

`--doc` document path (default `README.md`) · `--src` source dir (default `<docdir>/.hypedoc`) ·
`--out` image dir (default `<src>/img`) · `--scale` pixel ratio (default 2) ·
`--theme auto|light|dark|both` · `--opaque` keep white page background ·
`--browser` Chrome path (or `HYPEDOC_BROWSER`). Positional args select specific HTML files.
If no Chrome/Chromium is installed, a pinned Chromium is downloaded on first run.
