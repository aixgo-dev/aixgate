//go:build darwin

package jail

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// New returns a macOS Jailer that wraps the child process in `sandbox-exec`
// with a generated SBPL profile.
//
// Note (PRD §15.2 milestone 2): SBPL cannot hide the EXISTENCE of a file —
// it can only deny reads. So `ls ~/.ssh` will still list `id_rsa` on macOS;
// `cat ~/.ssh/id_rsa` will fail with "Operation not permitted". On Linux
// (FUSE) the file is invisible (ENOENT). We document this asymmetry rather
// than try to paper over it.
//
// Note (PRD §13.1): `sandbox-exec` is Apple-deprecated but still ships in
// every released macOS through 2026 and is used by Apple's own tooling.
// We use it intentionally for v0.1 — the v0.2+ path is FUSE-T.
func New(p Policy) (Jailer, error) {
	return &darwinJailer{policy: p}, nil
}

type darwinJailer struct {
	policy Policy
}

func (j *darwinJailer) Run(ctx context.Context, cmd string, args []string) error {
	profile, err := j.buildSBPL()
	if err != nil {
		return fmt.Errorf("build sandbox profile: %w", err)
	}

	// sandbox-exec wants the profile on disk. A temp file is fine — it
	// only has to outlive the child process, and the OS reaps it on
	// reboot if our cleanup misses.
	tmp, err := os.CreateTemp("", "aixgate-*.sb")
	if err != nil {
		return fmt.Errorf("create profile tempfile: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(profile); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write profile: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close profile: %w", err)
	}

	// `sandbox-exec -f profile.sb -- CMD ARGS...`
	// The `--` separator stops sandbox-exec's own arg parser so flags
	// belonging to CMD aren't misinterpreted.
	sbArgs := append([]string{"-f", tmp.Name(), "--", cmd}, args...)
	child := exec.CommandContext(ctx, "sandbox-exec", sbArgs...)
	child.Stdin = os.Stdin
	child.Stdout = os.Stdout
	child.Stderr = os.Stderr
	child.Env = os.Environ()

	if err := child.Run(); err != nil {
		// Forward the child's exit behaviour. *exec.ExitError carries
		// the exit code; the caller in cmd/aixgate translates that to
		// the process exit code so `aixgate run -- false` exits 1.
		return fmt.Errorf("sandbox-exec: %w", err)
	}
	return nil
}

// buildSBPL renders the Policy into an SBPL profile string.
//
// The profile is permissive by default (`(allow default)`) and adds explicit
// `(deny file-read* ...)` rules for the policy's deny list. This matches
// PRD §15.1.3 — v0.1 only validates the deny path; v0.2 flips the default.
func (j *darwinJailer) buildSBPL() (string, error) {
	var b strings.Builder
	b.WriteString("(version 1)\n")
	b.WriteString("(allow default)\n")
	b.WriteString("(deny file-read*\n")

	for _, pat := range j.policy.DenyReadGlobs {
		clause, err := sbplClauseForGlob(pat)
		if err != nil {
			return "", fmt.Errorf("translate %q: %w", pat, err)
		}
		b.WriteString("    ")
		b.WriteString(clause)
		b.WriteString("\n")
	}

	b.WriteString(")\n")
	return b.String(), nil
}

// sbplClauseForGlob converts a Go-style glob pattern to an SBPL match clause.
//
// We support two shapes:
//   - "**/<name>" or "**/<name>.*" — a basename match anywhere in the
//     filesystem. Rendered as a (regex ...) on the basename portion.
//   - Absolute path — rendered as a (literal "/abs/path") clause. The
//     caller is responsible for any ~ expansion before calling.
//
// Anything else is rejected to keep the v0.1 grammar small and auditable.
func sbplClauseForGlob(pattern string) (string, error) {
	// Absolute path — emit a literal clause.
	if filepath.IsAbs(pattern) {
		return fmt.Sprintf(`(literal %q)`, pattern), nil
	}

	// "**/<basename>" — match the basename anywhere.
	if rest, ok := strings.CutPrefix(pattern, "**/"); ok {
		// Translate '*' in the basename to a regex '.*'. Escape dots.
		// We deliberately do not support character classes or other
		// glob features in v0.1.
		escaped := regexEscapeGlobBasename(rest)
		// `^.*/` so the regex matches the path-separator before the
		// basename, anchoring the basename component.
		return fmt.Sprintf(`(regex #"^.*/%s$")`, escaped), nil
	}

	return "", fmt.Errorf("unsupported glob pattern (want absolute or **/<name>)")
}

// regexEscapeGlobBasename escapes regex metacharacters in a glob basename
// and translates `*` (glob) to `.*` (regex). Tight, deliberate, no surprises.
func regexEscapeGlobBasename(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '*':
			b.WriteString(`.*`)
		case '.', '+', '?', '(', ')', '[', ']', '{', '}', '|', '^', '$', '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
