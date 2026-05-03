//go:build linux

package jail

import "errors"

// New on Linux returns a stub. The FUSE-backed Linux implementation
// (hanwen/go-fuse + Landlock + seccomp-bpf, per PRD §6 and §14.1) is the
// v0.2 milestone; v0.1 ships macOS-only to validate the enforcement boundary
// on a single platform first (PRD §15.1.4).
//
// When the Linux backend lands, New will return an actual FUSE-mounting
// Jailer plus the mount-namespace child-isolation logic.
func New(_ Policy) (Jailer, error) {
	return nil, errors.New("aixgate: the Linux backend is not implemented in v0.1; the v0.1 PoC ships macOS-only (see docs/PRD.md §15). Build on macOS or wait for the v0.2 FUSE backend")
}
