// Package assets embeds the agent guide and the init starter template.
package assets

import _ "embed"

//go:embed GUIDE.md
var Guide string

//go:embed starter.html
var Starter string
