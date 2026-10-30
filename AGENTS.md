# AGENTS.md

Guidance for AI coding agents (Claude Code, Codex, Cursor, Copilot, …) working **on** this repo.
To **use** HypeDoc in another project, see [`skills/hypedoc/SKILL.md`](skills/hypedoc/SKILL.md) or run `hypedoc guide`.

## What this is

HypeDoc is a single Go binary. It screenshots self-contained HTML files with headless Chromium
and pins the PNGs to the top of a markdown document inside a managed block
(`<!-- hypedoc:start -->` … `<!-- hypedoc:end -->`). Everything outside the block is preserved byte for byte.

## Layout

| Path | Purpose |
|---|---|
| `main.go` | CLI: `build`, `init`, `strip`, `guide`, `version` (stdlib `flag`, no framework) |
| `internal/doc/` | Pure block inject/strip logic. Fully unit tested; no I/O |
| `internal/render/` | Chromium launch + capture via go-rod |
| `internal/assets/` | Embedded `GUIDE.md` (printed by `hypedoc guide`) and `starter.html` (used by `init`) |
| `.hypedoc/` | This repo's own README graphics (dogfood). `img/` is generated |
| `version.txt` | Single source of truth for the version; embedded into the binary |
| `.github/workflows/` | `ci.yml` (test + smoke on every push/PR), `release.yml` (release on `version.txt` change) |

## Commands

```sh
make build    # bin/hypedoc
make test     # gofmt check, go vet, go test
make smoke    # end-to-end init → build → strip in a temp dir (needs Chrome or network)
make readme   # regenerate this README's graphics from .hypedoc/*.html
```

Run `make test` before finishing any change. Run `make smoke` when touching `main.go` or `internal/render/`.

## Rules

- **Never hand-edit the README between the `hypedoc:start`/`hypedoc:end` markers.** Edit `.hypedoc/*.html`, then `make readme`.
- **Never create git tags or GitHub releases by hand.** Releases are cut by bumping `version.txt` (semver, strictly increasing) on `main`.
  Bump it only when asked to release: patch for fixes, minor for features, major for breaking CLI/marker changes.
- Keep dependencies minimal. go-rod is the only third-party dependency; prefer the standard library.
- `internal/doc` must stay pure (strings in, strings out) and covered by tests. Add a test for every behavior change.
- Markers are only recognized at the start of a line outside fenced code blocks, so docs can mention them safely. Keep it that way.
- Theme auto-detection keys on real `@media (prefers-color-scheme…)` rules or `matchMedia` calls, not prose. See `themedRe` in `main.go`.
- The CLI's text output goes to stderr; `--json` output goes to stdout and must stay valid JSON.
- Changes to flags, meta tags, or markers must be reflected in `internal/assets/GUIDE.md`, `README.md`, and `skills/hypedoc/SKILL.md`.

## README graphics style

`.hypedoc/*.html` use a neo-brutalist style: cream paper `#F3EFE6` with a faint `+` grid, plum `#4A2F3A` as the only
ink (text, borders, solid offset shadows), cyan `#44ADCB` accent used sparingly, stone `#DDD8CE`, chalk `#FBF9F4`.
Heavy Inter Display headlines, uppercase pill buttons, light theme only. Keep new graphics consistent.
