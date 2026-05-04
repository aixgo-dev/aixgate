package sandbox

import (
	"os"
	"path/filepath"
)

// Policy describes which paths the sandbox must hide from the child process.
//
// v0.1 has one Policy: the hardcoded V01Policy returned by DefaultV01Policy.
// In v0.2 this becomes a YAML-loaded structure (PRD §7).
type Policy struct {
	// DenyReadGlobs are file paths that must return ENOENT (Linux) or
	// access-denied (macOS) on read. Glob patterns are expanded relative
	// to the user's home and the working directory by the platform
	// backend; see darwin.go and linux.go.
	DenyReadGlobs []string

	// HomeDir is the user's home directory, used to expand `~/...` in
	// patterns. Captured at policy construction so tests can override it.
	HomeDir string
}

// DefaultV01Policy returns the hardcoded v0.1 policy from PRD §15.1.3:
// hide .env (and .env.*), SSH private keys, and AWS credentials.
//
// Anything not in the deny list passes through. This is intentional —
// v0.1 is deny-only-list to validate the enforcement boundary; v0.2
// flips to deny-by-default with an explicit allow list (PRD §11.2).
func DefaultV01Policy() Policy {
	home, err := os.UserHomeDir()
	if err != nil {
		// If we can't read $HOME, the caller almost certainly can't run
		// an agent meaningfully either. Fall back to an empty home so
		// the absolute paths below simply don't match — fail-open is
		// fine here because the worst case is "sandbox doesn't hide
		// what it claimed to hide" which the smoke test will catch.
		home = ""
	}
	return Policy{
		HomeDir: home,
		DenyReadGlobs: []string{
			// Any .env or .env.* under any working directory.
			// The platform backend expands the working-directory side.
			"**/.env",
			"**/.env.*",
			// SSH private keys — id_rsa, id_ed25519, id_ecdsa, etc.
			// id_*.pub is intentionally NOT denied: public keys are
			// safe to expose and agents legitimately read them.
			filepath.Join(home, ".ssh", "id_rsa"),
			filepath.Join(home, ".ssh", "id_ed25519"),
			filepath.Join(home, ".ssh", "id_ecdsa"),
			filepath.Join(home, ".ssh", "id_dsa"),
			// AWS shared credentials file.
			filepath.Join(home, ".aws", "credentials"),
		},
	}
}
