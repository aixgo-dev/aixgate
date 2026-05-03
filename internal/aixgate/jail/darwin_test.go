//go:build darwin

package jail

import (
	"strings"
	"testing"
)

// TestSBPLGeneration is the v0.1 minimal smoke check (PRD §15.4 forbids
// a real test suite; this just verifies the profile renders correctly).
func TestSBPLGeneration(t *testing.T) {
	p := Policy{
		HomeDir: "/Users/test",
		DenyReadGlobs: []string{
			"**/.env",
			"**/.env.*",
			"/Users/test/.ssh/id_rsa",
			"/Users/test/.aws/credentials",
		},
	}
	j := &darwinJailer{policy: p}
	profile, err := j.buildSBPL()
	if err != nil {
		t.Fatalf("buildSBPL: %v", err)
	}

	// The header is non-negotiable — sandbox-exec rejects malformed
	// profiles outright.
	for _, want := range []string{
		"(version 1)",
		"(allow default)",
		"(deny file-read*",
	} {
		if !strings.Contains(profile, want) {
			t.Errorf("profile missing %q. Profile:\n%s", want, profile)
		}
	}

	// Each policy entry must produce a clause. The exact rendering is
	// asserted by sbplClauseForGlob's unit tests; here we check counts.
	clauseCount := strings.Count(profile, "(regex ") + strings.Count(profile, "(literal ")
	if clauseCount != len(p.DenyReadGlobs) {
		t.Errorf("got %d clauses, want %d. Profile:\n%s",
			clauseCount, len(p.DenyReadGlobs), profile)
	}
}

func TestSbplClauseForGlob(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		// Absolute paths render as literals.
		{
			in:   "/Users/alice/.aws/credentials",
			want: `(literal "/Users/alice/.aws/credentials")`,
		},
		{
			in:   "/Users/alice/.ssh/id_rsa",
			want: `(literal "/Users/alice/.ssh/id_rsa")`,
		},
		// **/<basename> renders as a regex anchored on the basename.
		// Dots inside the basename must be escaped; * becomes .*.
		{
			in:   "**/.env",
			want: `(regex #"^.*/\.env$")`,
		},
		{
			in:   "**/.env.*",
			want: `(regex #"^.*/\.env\..*$")`,
		},
		// Anything that's neither absolute nor **/<name> is rejected.
		{in: "relative/path", wantErr: true},
		{in: ".env", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := sbplClauseForGlob(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
