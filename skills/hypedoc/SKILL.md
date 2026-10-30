---
name: hypedoc
description: Create high-fidelity graphics for a README or other markdown doc by designing self-contained HTML and rendering it with the hypedoc CLI. Use when asked to make a README look better, add a hero/banner/header image, feature graphics, or visual diagrams to markdown docs.
---

# HypeDoc

`hypedoc` screenshots HTML files into crisp 2x PNGs and pins them to the top of a markdown
document inside a managed block. Everything else in the document is preserved.

## Install (if `hypedoc` is not on PATH)

```sh
curl -fsSL https://raw.githubusercontent.com/stuyk/hypedoc/main/install.sh | sh
# or: go install github.com/stuyk/hypedoc@latest
```

## Workflow

1. Run `hypedoc guide` and follow its authoring rules exactly.
2. Read the target doc and the project it describes. Graphics must use real names, commands, and claims.
3. Create `.hypedoc/01-hero.html` (or run `hypedoc init --doc README.md`). Add more files as `02-…`, `03-…`; they render in filename order. 2–4 graphics is the sweet spot.
4. Run `hypedoc build --doc README.md --json` and read the report.
5. Look at every PNG listed in the report. Fix clipping, wrapping, overlap, or low contrast in the HTML and rebuild until they look right.

## Must not

- Edit anything between `<!-- hypedoc:start -->` and `<!-- hypedoc:end -->` by hand; it is overwritten on every build.
- Use remote fonts, remote images, `vh` units, or animations in the HTML.
- Run `hypedoc strip` unless asked to remove the graphics.
