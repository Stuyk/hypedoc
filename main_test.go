package main

import (
	"reflect"
	"testing"
)

func TestSchemesForAuto(t *testing.T) {
	cases := map[string][]string{
		`<style>@media (prefers-color-scheme: dark) { body {} }</style>`: {"light", "dark"},
		`<style>@media screen and (prefers-color-scheme:dark){}</style>`: {"light", "dark"},
		`<script>matchMedia('(prefers-color-scheme: dark)')</script>`:    {"light", "dark"},
		`<p>Use <code>prefers-color-scheme</code> for dark mode.</p>`:    {"light"},
		`<p>plain</p>`: {"light"},
	}
	for html, want := range cases {
		if got := schemesFor("auto", html); !reflect.DeepEqual(got, want) {
			t.Errorf("schemesFor(auto, %q) = %v, want %v", html, got, want)
		}
	}
}
