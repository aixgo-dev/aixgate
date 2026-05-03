//go:build linux

package jail

import (
	"context"
	"errors"
)

// New on Linux returns a stub. The FUSE-backed Linux implementation
// (hanwen/go-fuse + Landlock + seccomp-bpf, per PRD §6 and §14.1) is the
// v0.2 milestone; v0.1 ships macOS-only to validate the enforcement boundary
// on a single platform first (PRD §15.1.4).
//
// When the Linux backend lands, this file's New will return an actual
// FUSE-mounting Jailer and the mount-namespace child-isolation logic.
func New(_ Policy) (Jailer, error) {
	return nil, errors.New("aixgate: the Linux backend is not implemented in v0.1; the v0.1 PoC ships macOS-only (see docs/PRD.md §15). Build on macOS or wait for the v0.2 FUSE backend")
}

// linuxJailerStub exists only to give the package a concrete type on Linux
// builds, so future references compile while New returns the error above.
// It will be replaced by the real FUSE jailer in v0.2.
type linuxJailerStub struct{}

func (linuxJailerStub) Run(_ context.Context, _ string, _ []string) error {
	return errors.New("aixgate: linux backend not implemented in v0.1")
}
