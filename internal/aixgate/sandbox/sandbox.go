// Package sandbox provides the platform-agnostic enforcement boundary for aixgate.
//
// v0.1 scope: a single hardcoded policy that hides .env files, SSH private
// keys, and AWS credentials from the child process. The Sandbox interface
// has one method (Run) so each platform's enforcement primitive — Landlock +
// FUSE on Linux, sandbox-exec on macOS — can implement it independently.
//
// See docs/PRD.md §6 (Architecture) and §15 (v0.1 Weekend Build Plan) for
// scope, threat model, and platform asymmetries.
package sandbox

import "context"

// Sandbox launches a command inside a deny-by-default boundary configured by
// the embedded Policy. Implementations are platform-specific and selected at
// build time via Go build tags.
//
// Run blocks until the child process exits and returns the child's exit
// behaviour as an error: nil on a clean exit (status 0), or an error wrapping
// *exec.ExitError on a non-zero exit. Errors related to setting up the
// sandbox itself (writing the SBPL profile, creating the FUSE mount, etc.)
// are returned before the child is spawned and are wrapped with %w so
// callers can use errors.Is and errors.As.
type Sandbox interface {
	Run(ctx context.Context, cmd string, args []string) error
}

// New returns the platform-specific Sandbox for the current OS, configured
// with the supplied policy. On Linux the FUSE-backed implementation is not
// yet available in v0.1 and New returns an error directing the user to
// macOS for the PoC.
//
// The actual constructor lives in darwin.go / linux.go behind build tags.
