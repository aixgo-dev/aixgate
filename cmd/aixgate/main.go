// Package main is the entry point for the aixgate CLI.
//
// Aixgate is a deny-by-default sandbox for AI coding agents. The v0.1
// implementation is in flight per docs/PRD.md §15; this file is a
// placeholder so go.mod stays consistent and CI has something to lint.
package main

import (
	"fmt"

	// Reserved consumption of aixgo's public API — aixgate will use
	// pkg/security helpers in v0.1 (input validation, SSRF protection).
	// The blank import keeps go.mod honest until the real call sites land.
	_ "github.com/aixgo-dev/aixgo/pkg/security"
)

// Version is set at build time by GoReleaser via -ldflags.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func main() {
	fmt.Printf("aixgate %s (commit %s, built %s)\n", Version, Commit, Date)
	fmt.Println("Aixgate v0.1 is in active development. See https://github.com/aixgo-dev/aixgate/blob/main/docs/PRD.md")
}
