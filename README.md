# Aixgate

**A deny-by-default sandbox for AI coding agents.**

Your `.env`, `~/.aws`, and `~/.ssh` stay invisible — even when Claude Code, Cursor, Aider, or Codex go off-script. One binary. Pure Go. macOS and Linux.

[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](LICENSE)
[![Go Reference](https://img.shields.io/badge/go.dev-reference-007D9C?style=flat-square&logo=go)](https://pkg.go.dev/github.com/aixgo-dev/aixgate)
[![Platform: macOS | Linux](https://img.shields.io/badge/platform-macOS%20%7C%20Linux-lightgrey?style=flat-square)](#platform-support)
[![Status: pre-1.0](https://img.shields.io/badge/status-pre--1.0-orange?style=flat-square)](#status)
[![GitHub stars](https://img.shields.io/github/stars/aixgo-dev/aixgate?style=flat-square)](https://github.com/aixgo-dev/aixgate/stargazers)

> **Status: pre-v0.1.** Aixgate is in active development. The v0.1 weekend build is in flight per [PRD §15](docs/PRD.md). Star this repo to follow the v0.1 release.

---

## The 10-second demo

```bash
# Without Aixgate, an AI agent can read everything you can:
cat .env
# OPENAI_API_KEY=sk-...
# DATABASE_URL=postgres://...

# With Aixgate, sensitive paths simply do not exist as far as the agent is concerned:
aixgate run -- cat .env
# cat: .env: No such file or directory

aixgate run -- ls ~/.ssh
# (empty)

aixgate audit tail --denied-only
# 2026-05-03T10:14:22Z DENY  open    /Users/you/.env       cat
# 2026-05-03T10:14:25Z DENY  readdir /Users/you/.ssh      ls
```

That's the whole product, rendered in three commands.

---

## Why Aixgate

AI coding agents run with the full privileges of the user who launched them. That means your `.env` files, `~/.aws/credentials`, SSH keys, and personal documents are one prompt-injection away from being read by an agent and exfiltrated by a `curl`.

Aixgate is a vendor-agnostic OS sandbox that wraps any AI coding agent — Claude Code, Cursor, Aider, OpenAI Codex, your own — in a deny-by-default filesystem and exec policy. Sensitive paths return `ENOENT`, so the agent never knows they exist. Every access decision is logged.

One binary. Pure Go. No Docker, no daemon, no kernel extension.

### How it compares

| | Aixgate | gVisor | bubblewrap | firejail | Docker Desktop | Vendor permission prompts |
|---|---|---|---|---|---|---|
| Targets developer laptops | ✅ | ❌ servers | ⚠️ Linux only | ⚠️ Linux only | ⚠️ heavy | ✅ |
| macOS support | ✅ | ❌ | ❌ | ❌ | ✅ | ✅ |
| Vendor-agnostic | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ per-vendor |
| Hides path existence (`ENOENT`) | ✅ Linux | ❌ | ⚠️ | ⚠️ | ❌ | ❌ |
| Declarative portable policy | ✅ | ❌ | ❌ | ⚠️ | ❌ | ❌ |
| Single binary, no daemon | ✅ | ❌ | ✅ | ✅ | ❌ | ❌ |
| Tamper-resistant audit log | ✅ | ❌ | ❌ | ❌ | ❌ | ⚠️ |

Aixgate is built specifically for the developer-laptop + AI-coding-agent threat model. The other tools are excellent at what they do — they just don't do this.

---

## How it works

On Linux, Aixgate composes [Landlock](https://landlock.io/) (filesystem ABI), [seccomp-bpf](https://www.kernel.org/doc/html/latest/userspace-api/seccomp_filter.html) (syscall filtering), and [Go-FUSE](https://github.com/hanwen/go-fuse) (overlay/mount-time path hiding) to enforce a deny-by-default policy at the OS boundary, before any syscall reaches the kernel's normal authorisation path.

On macOS, Aixgate generates a [`sandbox-exec`](https://www.unix.com/man-page/osx/1/sandbox-exec/) profile from the same policy and launches the agent inside it. This is Apple's older sandboxing primitive — deprecated-but-shipped — and we use it transparently. ([Platform asymmetry note](#platform-asymmetry-note) below.)

Either way, you launch your AI coding agent the same way:

```bash
aixgate run --profile claude-code -- claude
```

See [`docs/PRD.md`](docs/PRD.md) §6 for the full architecture.

---

## Quick start

### Install

**macOS (Homebrew)** — *available at v0.1*:

```bash
brew install aixgo-dev/tap/aixgate
```

**Linux (one-liner)** — *available at v0.1*:

```bash
curl -sSL https://aixgo.dev/aixgate/install.sh | sh
```

**From source (any platform)**:

```bash
go install github.com/aixgo-dev/aixgate/cmd/aixgate@latest
```

### Run an agent in a sandbox

```bash
cd ~/code/my-project

# Use a built-in profile
aixgate run --profile claude-code -- claude

# Or define your own policy
cat > .aixgate.yaml <<'YAML'
allow:
  - read: ./
  - write: ./
  - exec: [git, go, npm, pnpm]
deny:
  - .env
  - ~/.ssh
  - ~/.aws
  - ~/Documents
YAML
aixgate run -- claude
```

### Inspect what was blocked

```bash
aixgate audit tail            # live tail
aixgate audit query --since 1h --denied-only
```

---

## Profiles

Aixgate ships with curated profiles for popular AI coding agents:

- `claude-code` — Anthropic's Claude Code CLI
- `cursor` — Cursor's command-line agent mode
- `aider` — paul-gauthier/aider
- `codex` — OpenAI Codex agents
- `generic` — any process; conservative defaults

Profiles live in [`profiles/`](profiles/) and are versioned with the binary. Override or extend them via `.aixgate.yaml` in your project root. See [`docs/PRD.md`](docs/PRD.md) §7 for the policy schema.

---

## Security model

### What Aixgate defends against

- **Prompt injection** that causes an agent to read or write outside its declared scope.
- **Autonomous-run drift** where an agent operating with reduced human oversight makes unintended filesystem changes.
- **Compliance-driven access controls** for developers operating under SOC 2 / ISO 27001 / financial regulation.

### What Aixgate does **not** defend against

- **Rooted hosts.** Aixgate does not defend against root-level attackers. If an attacker has root, your sandboxing assumptions are gone (PRD §3.3).
- **Kernel exploits.** Landlock and seccomp escapes are kernel CVEs, not Aixgate vulnerabilities.
- **Vulnerabilities in the AI agent itself.** A bug in Claude Code is upstream's problem.

### Platform asymmetry note

`sandbox-exec` on macOS **cannot hide file existence** — it returns `EACCES` (permission denied) rather than `ENOENT` (no such file). On Linux, the Landlock + Go-FUSE composition returns `ENOENT` for denied reads. This is a documented platform asymmetry (PRD §15.2 Milestone 2), not a bug. If you need uniform `ENOENT` behaviour today, run on Linux.

`sandbox-exec` is also Apple-deprecated. We continue to use it transparently because Apple still ships and signs it; if Apple removes it, our v1.x roadmap (PRD §11.4) covers the migration to alternatives.

### Reporting vulnerabilities

See [SECURITY.md](SECURITY.md). Use [GitHub Security Advisories](https://github.com/aixgo-dev/aixgate/security/advisories/new) for private disclosure.

### Pre-v0.1 caveat

Until v0.2 ships and stabilises, **do not rely on Aixgate as your sole control**. Treat it as defence-in-depth, not a primary boundary.

---

## Roadmap

| Version | Targets |
|---|---|
| **v0.1** *(in flight)* | macOS + Linux, Claude Code / Aider / Cursor / Codex profiles, deny-by-default file policy, append-only audit log |
| **v0.2** | Network egress proxy, exec allowlist, policy template marketplace, first-party Homebrew tap |
| **v1.0** | Stable policy schema, signed audit log, FUSE-T fallback for macOS, comprehensive profile library |
| **v1.x** | Windows (WFP), eBPF backend on Linux, fleet management hook for `aixgo-cloud` |

Full roadmap: [PRD §11](docs/PRD.md).

---

## Status

**Pre-v0.1.** No Go source code yet — this repository contains the [PRD](docs/PRD.md), governance scaffolding, CI workflows, and ADR sequence. The v0.1 weekend build is in flight per [PRD §15](docs/PRD.md).

★ Star this repo to follow the v0.1 release.

---

## Contributing

Aixgate accepts contributions under a Developer Certificate of Origin (DCO). Sign your commits with `git commit -s`.

Open issues for discussion before large changes. Good first contributions: profile additions for new agents, documentation improvements, integration tests for specific AI coding agent workflows.

---

## See also

- [aixgo](https://github.com/aixgo-dev/aixgo) — The Go framework for building AI agents that Aixgate runs safely.
- [aixgo.dev](https://aixgo.dev) — Documentation and guides for the aixgo-dev family.

---

## License

[MIT](LICENSE).

> aixgo.dev builds agents. Aixgate keeps them in their lane.
