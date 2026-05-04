# CLAUDE.md - AI Assistant Project Guide

**Last Updated**: 2026-05-03
**Target**: AI assistants (Claude Code, GitHub Copilot, Cursor, etc.)
**Current release**: see [GitHub Releases](https://github.com/aixgo-dev/aixgate/releases/latest) (canonical version is the git tag)

Quick reference for AI assistants working with Aixgate — a runtime sandbox for AI coding agents.

## Table of Contents

- [Project Overview](#project-overview)
- [Architecture](#architecture)
- [Code Conventions](#code-conventions)
- [Key Concepts](#key-concepts)
- [Common Tasks](#common-tasks)
- [Quick Reference](#quick-reference)

---

## Project Overview

Aixgate is a **runtime sandbox for AI coding agents** (Claude Code, Cursor, Aider, OpenAI Codex agents, anything that runs as a process on behalf of an LLM). It enforces deny-by-default filesystem and exec policies at the OS boundary so sensitive files like `.env`, `~/.aws`, and `~/.ssh` are not visible to the agent unless an explicit policy rule permits it.

**Installation**: `go install github.com/aixgo-dev/aixgate/cmd/aixgate@latest`

### Why Aixgate

| Threat | Aixgate behaviour |
|--------|-------------------|
| Prompt injection reads `.env` | `ENOENT` — agent can't see the file exists |
| Autonomous run writes outside project | Write denied at syscall boundary |
| Agent shells out to `curl` for exfiltration | Exec allowlist blocks unauthorised commands |
| Compliance audit needed | Append-only audit log of every access decision |

### Target Users

- **Developers running AI coding agents** — Claude Code, Cursor, Aider, OpenAI Codex
- **Security-conscious engineers** under SOC 2 / ISO 27001 / regulated environments
- **Consultants** working in multiple client repos who need credential isolation
- **Teams** running long autonomous agent jobs and wanting a recorded audit trail

### Current Status

- **Maturity**: Pre-v0.1. Repository contains PRD + scaffolding; v0.1 implementation is in flight per [PRD §15](docs/PRD.md).
- **Go Version**: 1.26+
- **Platforms**: macOS (Apple Silicon + Intel) and Linux (x86_64 + arm64). Windows is on the v1.x roadmap (PRD §11.4).
- **License**: MIT

### Development Focus

Current priorities (per PRD §15 weekend build plan):

1. **`cmd/aixgate run`** — launch a child process inside a deny-by-default sandbox.
2. **Linux backend** — Landlock + seccomp-bpf + Go-FUSE composition.
3. **macOS backend** — `sandbox-exec` profile generation and launch.
4. **Policy parser** — `.aixgate.yaml` schema.
5. **Audit log** — append-only JSONL with timestamp, action, path, command.

### Reference Documents

**MANDATORY**: See [**docs/PRD.md**](docs/PRD.md) for the complete product requirements document — the source of truth for scope, threat model, architecture, policy schema, and roadmap.

---

## Architecture

### Layer Overview

```text
                  Aixgate CLI (cmd/aixgate)
                          ↓
              Policy Parser (.aixgate.yaml → AST)
                          ↓
              Sandbox Runtime (internal/aixgate/sandbox)
                ↓                          ↓
         Linux backend                macOS backend
   (Landlock + seccomp + FUSE)      (sandbox-exec profile)
                ↓                          ↓
         Audited child process (the AI coding agent)
                          ↓
                    Audit Log (JSONL)
```

### Repository Layout (target — most paths do not exist yet)

```
cmd/aixgate/                # CLI entry point (run, audit, profiles, version)
pkg/aixgate/                # Public client library (for aixgo framework integration:
                            #   AIXGATE_ACTIVE=1 detection, audit-context streaming)
internal/aixgate/
  ├── sandbox/              # platform-agnostic sandbox orchestration (Sandbox interface, Policy)
  ├── policy/               # .aixgate.yaml parser + AST + validation (v0.2)
  ├── fs/                   # FUSE overlay (Linux), path masking (v0.2)
  ├── audit/                # JSONL audit log writer + tail/query (v0.2)
  └── profiles/             # built-in profiles (claude-code, cursor, aider, codex, generic) (v0.2)
profiles/                   # YAML profile files shipped with the binary
docs/
  ├── PRD.md                # Product requirements document
  └── adr/                  # Architecture decision records
```

### Dependency Direction

Aixgate **imports from** [`github.com/aixgo-dev/aixgo`](https://github.com/aixgo-dev/aixgo) for shared infrastructure:

- `pkg/security` — input validation, SSRF protection
- `pkg/observability` — health checks, metrics

Aixgate **never imports from** [`github.com/charlesgreen/atlas`](https://github.com/charlesgreen/atlas) (closed-core; out of dependency tree). Aixgo never imports from Aixgate. This is the open-core boundary documented in [aixgo ADR 0002](https://github.com/aixgo-dev/aixgo/blob/main/docs/adr/0002-aixgate-separate-repo.md).

---

## Code Conventions

Mirror aixgo's conventions. The 30-second version:

### Naming

```go
// Interfaces: noun
type Sandbox interface { ... }
type PolicyEnforcer interface { ... }

// Constructors: New* prefix
func NewSandbox(cfg Config) *Sandbox

// Errors: Err* prefix or sentinel
var ErrPolicyViolation = errors.New("policy violation")

// Booleans: Is*/Has*/Can* prefix
func (p *Policy) AllowsRead(path string) bool
```

### Error Handling

```go
// Always wrap with %w
result, err := sb.Run(ctx, cmd, args)
if err != nil {
    return nil, fmt.Errorf("run sandbox: %w", err)
}

// Sentinel errors for category checks
var (
    ErrPolicyViolation = errors.New("policy violation")
    ErrSandboxEscape   = errors.New("sandbox integrity check failed")
    ErrAuditWrite      = errors.New("audit log write failed")
)
```

### Context

```go
// Always first parameter
func (s *Sandbox) Run(ctx context.Context, cmd string, args []string) error

// Use for timeouts and cancellation
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()
```

### Concurrency

The sandbox runtime is single-threaded per child process; concurrency lives in the audit log writer (which fans in events from multiple sandboxes).

```go
// Audit log uses a fan-in channel pattern
type AuditWriter struct {
    events chan Event
    file   *os.File
}
```

### Testing

- **Unit tests** for the policy parser, AST, and audit log format. Pure Go, fast.
- **Integration tests** behind build tags: `//go:build linux_landlock` for Landlock-specific paths; `//go:build darwin` for `sandbox-exec` paths.
- **Conformance tests**: a shared suite (`tests/conformance/`) that runs against every backend and verifies "given policy P and operation O, the outcome is the same regardless of backend."

```go
//go:build linux_landlock
package fs_test

func TestLandlock_DenyEnv(t *testing.T) {
    // Requires Linux 5.13+ and Landlock-enabled kernel.
    if !landlock.Supported() {
        t.Skip("Landlock not supported on this kernel")
    }
    // ...
}
```

### Security Patterns

- **Validate every path against the policy AST before passing to syscalls.**
- **Never trust paths from the child process** — they may be controlled by an injected prompt.
- **Use `syscall.Faccessat2` with `AT_SYMLINK_NOFOLLOW`** to prevent symlink-based escapes (Linux).
- **Default to denying writes outside `cwd`** unless the policy explicitly allows.
- **Audit-log every decision** — both ALLOW and DENY — so post-hoc review is possible.

### Example Secrets in Documentation

Same conventions as aixgo:

```yaml
# GOOD - placeholder patterns
OPENAI_API_KEY=<your-openai-api-key>
ANTHROPIC_API_KEY=<your-anthropic-api-key>

# GOOD - test fixtures
apiKey := "test-fixture-not-a-real-key-1"
```

The `.gitleaks.toml` config allowlists these patterns.

---

## Key Concepts

### Policy Schema

Policies are YAML, parsed into an AST, and validated before the sandbox starts. Schema is versioned via top-level `apiVersion`. See [PRD §7](docs/PRD.md) for the full schema; example:

```yaml
apiVersion: aixgate/v1
kind: Policy
allow:
  - read: ./
  - write: ./
  - read: ~/.gitconfig
  - exec: [git, go, npm, pnpm, claude]
deny:
  - .env
  - .env.*
  - ~/.ssh/**
  - ~/.aws/**
  - ~/Documents/**
  - **/node_modules/.cache/**
network:
  egress:
    allow: [api.anthropic.com, api.openai.com, github.com]
    deny: ["*"]
```

### Sandbox Lifecycle

1. **Compile** policy AST.
2. **Spawn** child process with platform-specific isolation primitives applied (Landlock + seccomp + FUSE on Linux; `sandbox-exec` profile on macOS).
3. **Audit** every relevant syscall via the platform-specific audit hook.
4. **Wait** for child to exit; rotate audit log; emit summary.

### Audit Log Format

Append-only JSONL. One record per access decision. See [PRD §8](docs/PRD.md) for full schema.

```jsonl
{"ts":"2026-05-03T10:14:22.491Z","sandbox":"sb-abc123","decision":"DENY","action":"open","path":"/Users/you/.env","cmd":"cat","pid":42137,"reason":"matched deny rule .env"}
{"ts":"2026-05-03T10:14:25.108Z","sandbox":"sb-abc123","decision":"ALLOW","action":"read","path":"/Users/you/code/my-project/main.go","cmd":"claude","pid":42138}
```

### Built-in Profiles

Profiles in `profiles/<name>.yaml` ship as embedded files (`//go:embed`). Users can override via `~/.aixgate/profiles/<name>.yaml` or per-project `.aixgate.yaml`.

### Backend Abstraction

```go
package sandbox

type Backend interface {
    Name() string                                    // "linux-landlock" | "darwin-sandbox-exec"
    Supported() (bool, error)                        // does this host support this backend?
    Compile(p *Policy) (CompiledPolicy, error)       // backend-specific compilation
    Spawn(ctx context.Context, c *exec.Cmd, cp CompiledPolicy) (Sandbox, error)
}
```

The CLI selects the backend automatically based on `runtime.GOOS`.

---

## Common Tasks

### Add a new built-in profile

1. Create `profiles/<agent>.yaml` with deny/allow rules tailored to the agent's known behaviour.
2. Embed it in `internal/aixgate/profiles/embed.go` via `//go:embed`.
3. Add a conformance test in `tests/conformance/profiles_test.go` exercising a representative workflow.
4. Document it in [`docs/PRD.md`](docs/PRD.md) §7.4 and the README profiles table.

### Add a new policy primitive

1. Extend the AST in `internal/aixgate/policy/ast.go`.
2. Add parser support in `internal/aixgate/policy/parser.go`.
3. Implement enforcement in **every** backend (`internal/aixgate/sandbox/linux.go`, `internal/aixgate/sandbox/darwin.go`).
4. Add unit tests for the parser and a conformance test that runs against every backend.
5. Document in [PRD §7](docs/PRD.md).

### Write a sandbox integration test

```go
//go:build linux
package conformance_test

func TestDenyEnv_ENOENT(t *testing.T) {
    j := newTestJail(t, defaultPolicy)
    out, err := j.RunCmd("cat", ".env")
    if !errors.Is(err, syscall.ENOENT) {
        t.Fatalf("expected ENOENT, got %v (output: %s)", err, out)
    }
}
```

Conformance tests should pass on every backend so platform asymmetries are explicit.

### Add a new ADR

ADRs live in `docs/adr/` and are numbered `0001-`, `0002-`, ... Independent from aixgo's ADR sequence.

```bash
N=$(printf "%04d" $(($(ls docs/adr/ | grep -c '^[0-9]') + 1)))
cp docs/adr/template.md docs/adr/$N-short-title.md
```

(Template lives at `docs/adr/template.md` once created.)

---

## Quick Reference

### Commands

```bash
make build          # Build the aixgate binary into dist/
make test           # Run tests with race detection
make lint           # Run golangci-lint
make coverage       # Generate HTML coverage report
make install        # Install via go install
make check          # fmt + vet + lint + test
```

### Import Paths

```go
import (
    "github.com/aixgo-dev/aixgate"

    // Internal (closed to external consumers)
    "github.com/aixgo-dev/aixgate/internal/aixgate/policy"
    "github.com/aixgo-dev/aixgate/internal/aixgate/sandbox"
    "github.com/aixgo-dev/aixgate/internal/aixgate/audit"

    // Public client library (for aixgo framework integration)
    "github.com/aixgo-dev/aixgate/pkg/aixgate"

    // Shared from aixgo framework
    "github.com/aixgo-dev/aixgo/pkg/security"
    "github.com/aixgo-dev/aixgo/pkg/observability"
)
```

### Key Files (target — most do not exist yet)

- **CLI**: `cmd/aixgate/main.go`, `cmd/aixgate/cmd/{run,audit,profiles,version}.go`
- **Policy**: `internal/aixgate/policy/{parser,ast,validate}.go`
- **Sandbox**: `internal/aixgate/sandbox/{sandbox,linux,darwin,policy}.go`
- **Audit**: `internal/aixgate/audit/{writer,query,tail}.go`
- **Profiles**: `internal/aixgate/profiles/embed.go`, `profiles/*.yaml`

### Documentation

- [**docs/PRD.md**](docs/PRD.md) — **Authoritative product requirements** (update on scope/threat-model changes)
- [docs/adr/](docs/adr/) — Architecture decision records (independent sequence from aixgo)
- [SECURITY.md](SECURITY.md) — Vulnerability reporting
- [CHANGELOG.md](CHANGELOG.md) — Release notes

### Resources

- Website: <https://aixgo.dev>
- GitHub: <https://github.com/aixgo-dev/aixgate>
- Sibling repo (framework): <https://github.com/aixgo-dev/aixgo>
- Discussions: <https://github.com/orgs/aixgo-dev/discussions>

> aixgo.dev builds agents. Aixgate keeps them in their lane.
