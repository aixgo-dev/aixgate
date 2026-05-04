// Package main is the entry point for the aixgate CLI.
//
// v0.1 ships a single subcommand, `aixgate run`, that wraps a child
// process in a deny-by-default sandbox using the hardcoded policy from
// internal/aixgate/sandbox.DefaultV01Policy. See docs/PRD.md §15.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/aixgo-dev/aixgate/internal/aixgate/sandbox"
)

// Build-time variables populated by GoReleaser via -ldflags. See .goreleaser.yaml.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		// Cobra has already printed user-facing error output.
		// Translate exec exit codes faithfully so `aixgate run -- false`
		// exits 1 rather than 0 or some opaque cobra default.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "aixgate",
		Short: "A deny-by-default sandbox for AI coding agents",
		Long: `aixgate launches a child process inside a deny-by-default sandbox
that hides .env files, SSH private keys, and AWS credentials from the
process and any subprocesses it spawns. v0.1 ships macOS only and uses
a hardcoded policy.

See https://github.com/aixgo-dev/aixgate/blob/main/docs/PRD.md for the
full design.`,
		SilenceUsage:  true,
		SilenceErrors: false,
	}
	root.AddCommand(newRunCmd(), newVersionCmd())
	return root
}

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run -- CMD [ARGS...]",
		Short: "Run a command inside the sandbox",
		Long: `Run launches CMD ARGS inside a sandbox configured with aixgate's v0.1
hardcoded policy. The standard input, output, and error of the child are
inherited; the child's exit code is propagated.

Example:

  aixgate run -- claude              # launch Claude Code in the sandbox
  aixgate run -- ls ~/.ssh           # ls inside the sandbox
  aixgate run -- bash -c "cat .env"  # subprocesses are also sandboxed`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			policy := sandbox.DefaultV01Policy()
			sb, err := sandbox.New(policy)
			if err != nil {
				return err
			}

			// Forward SIGINT/SIGTERM to the child via the cancellable
			// context. exec.CommandContext sends SIGKILL on cancel,
			// which is correct behaviour for a sandboxed child that
			// has gone unresponsive.
			ctx, cancel := signal.NotifyContext(
				context.Background(), syscall.SIGINT, syscall.SIGTERM,
			)
			defer cancel()

			return sb.Run(ctx, args[0], args[1:])
		},
	}
	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print build information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("aixgate %s (commit %s, built %s)\n", Version, Commit, Date)
		},
	}
}
