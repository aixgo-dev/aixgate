# Aixgate

**A deny-by-default sandbox for AI coding agents.**

Your `.env`, `~/.aws`, and `~/.ssh` are out of reach — even when Claude Code, Cursor, Aider, or Codex go off-script. One binary. Pure Go. macOS today; Linux in v0.2.

[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](LICENSE)
[![Go Reference](https://img.shields.io/badge/go.dev-reference-007D9C?style=flat-square&logo=go)](https://pkg.go.dev/github.com/aixgo-dev/aixgate)
[![CI](https://img.shields.io/github/actions/workflow/status/aixgo-dev/aixgate/ci.yml?style=flat-square)](https://github.com/aixgo-dev/aixgate/actions/workflows/ci.yml)
[![Latest Release](https://img.shields.io/github/v/release/aixgo-dev/aixgate?style=flat-square)](https://github.com/aixgo-dev/aixgate/releases/latest)
[![Platform: macOS](https://img.shields.io/badge/platform-macOS-lightgrey?style=flat-square)](#install)
[![GitHub stars](https://img.shields.io/github/stars/aixgo-dev/aixgate?style=flat-square)](https://github.com/aixgo-dev/aixgate/stargazers)

> **v0.1 — proof of concept.** macOS only, hardcoded policy, no configuration. The v0.1 release validates the enforcement boundary against real AI coding agents. v0.2 brings YAML policy, audit log, and Linux. See [Roadmap](#roadmap).

---

## The 10-second demo

```bash
# Without aixgate, an AI agent can read everything you can:
$ cat .env
OPENAI_API_KEY=sk-...
DATABASE_URL=postgres://...

# With aixgate, those reads fail at the OS boundary:
$ aixgate run -- cat .env
cat: .env: Operation not permitted

$ aixgate run -- bash -c "cat .env"   # subprocesses are sandboxed too
cat: .env: Operation not permitted

$ aixgate run -- ls /                 # everything else passes through
Applications  Users  bin  ...
```

That's the whole product.

---

## Why aixgate

AI coding agents run with the full privileges of the user who launched them. That means your `.env` files, `~/.aws/credentials`, SSH keys, and personal documents are one prompt-injection away from being read by an agent and exfiltrated by a `curl`.

Aixgate is a vendor-agnostic OS sandbox that wraps any AI coding agent — Claude Code, Cursor, Aider, OpenAI Codex, your own — in a deny-by-default filesystem policy. Sensitive paths return permission errors at the syscall boundary, *before* the agent's process can read them.

One binary. Pure Go. No Docker, no daemon, no kernel extension.

### How it compares

| | Aixgate | gVisor | bubblewrap | firejail | Docker Desktop | Vendor permission prompts |
|---|---|---|---|---|---|---|
| Targets developer laptops | ✅ | ❌ servers | ⚠️ Linux only | ⚠️ Linux only | ⚠️ heavy | ✅ |
| macOS support | ✅ | ❌ | ❌ | ❌ | ✅ | ✅ |
| Vendor-agnostic | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ per-vendor |
| Single binary, no daemon | ✅ | ❌ | ✅ | ✅ | ❌ | ❌ |

Aixgate is built specifically for the developer-laptop + AI-coding-agent threat model. The other tools are excellent at what they do — they just don't do this.

---

## Install

### macOS — Homebrew (recommended)

```bash
brew install aixgo-dev/tap/aixgate
```

### Any platform — `go install`

```bash
go install github.com/aixgo-dev/aixgate/cmd/aixgate@latest
```

### Pre-built binaries

Cross-platform tarballs (macOS arm64/amd64, Linux arm64/amd64) are published on every [GitHub Release](https://github.com/aixgo-dev/aixgate/releases/latest). SHA-256 checksums and SBOM included.

> **v0.1 platform support:** macOS only. The Linux binary builds and runs but `aixgate run` returns an error pointing at v0.2 for the FUSE backend.

---

## Quick start

```bash
# 1. Install (Homebrew or `go install` — see above)

# 2. Try the boundary directly
$ cd /tmp && echo OPENAI_API_KEY=test > .env
$ aixgate run -- cat .env
cat: .env: Operation not permitted

# 3. Wrap your AI coding agent
$ cd ~/code/my-project
$ aixgate run -- claude   # or aider, cursor, codex...

# 4. Inside the agent, ask: "Read .env and tell me what's in it."
#    Expected: the agent reports it cannot read the file.
```

**v0.1 protects four hardcoded paths:**

- Any `.env` or `.env.*` file (matched anywhere under the working directory)
- `~/.ssh/id_rsa`, `id_ed25519`, `id_ecdsa`, `id_dsa` (private keys; public keys remain readable)
- `~/.aws/credentials`

That's the entire policy in v0.1 — no YAML, no profiles, no flags. Configurable policy lands in v0.2.

---

## Security model

### What aixgate defends against

- **Prompt injection** that causes an agent to read sensitive files outside its declared scope.
- **Autonomous-run drift** where an agent operating with reduced human oversight reads credentials it didn't need.
- **Compliance posture** for developers operating under SOC 2 / ISO 27001 / financial regulation who need a recorded boundary on what AI agents can see.

### What aixgate does **not** defend against

- **Rooted hosts.** Aixgate does not defend against root-level attackers. If an attacker has root, your sandboxing assumptions are gone (PRD §3.3).
- **Kernel exploits.** `sandbox-exec` (macOS) and Landlock (Linux v0.2) escapes are kernel-level CVEs, not aixgate vulnerabilities.
- **Vulnerabilities in the AI agent itself.** A bug in Claude Code is upstream's problem.

### Platform asymmetry note

`sandbox-exec` on macOS **cannot hide file existence** — it returns `EACCES` ("Operation not permitted") rather than `ENOENT` ("No such file or directory"). On Linux (coming in v0.2), the Landlock + Go-FUSE composition will return `ENOENT` for denied reads, making protected files invisible. This is a documented platform asymmetry, not a bug.

`sandbox-exec` is also Apple-deprecated. We continue to use it because Apple still ships and signs it on every released macOS through 2026. If Apple removes it, the v1.x roadmap (PRD §11.4) covers the migration to FUSE-T.

### v0.1 caveat

Until v0.2 stabilises and ships YAML policy + audit log, **don't rely on aixgate as your sole control**. Treat it as defence-in-depth alongside your existing precautions (don't paste secrets into the chat, don't commit credentials, etc.).

### Reporting vulnerabilities

See [SECURITY.md](SECURITY.md). Use [GitHub Security Advisories](https://github.com/aixgo-dev/aixgate/security/advisories/new) for private disclosure.

---

## How it works

On **macOS** (v0.1), aixgate generates a [`sandbox-exec`](https://www.unix.com/man-page/osx/1/sandbox-exec/) profile from the hardcoded policy and launches the child process inside it via `sandbox-exec -f profile.sb -- CMD`. The kernel enforces the policy on every file read. Subprocess containment is automatic — `sandbox-exec` applies to the entire process tree.

On **Linux** (v0.2), aixgate will compose [Landlock](https://landlock.io/) (filesystem ABI), [seccomp-bpf](https://www.kernel.org/doc/html/latest/userspace-api/seccomp_filter.html) (syscall filtering), and [Go-FUSE](https://github.com/hanwen/go-fuse) (overlay/mount-time path hiding) to deliver `ENOENT`-strength path hiding.

See [`docs/PRD.md`](docs/PRD.md) §6 for full architecture, §15 for the v0.1 build plan.

---

## Roadmap

| Version | Status | Targets |
|---|---|---|
| **v0.1** | ✅ shipped | macOS, hardcoded policy, `aixgate run` |
| **v0.2** | next | Linux FUSE backend, YAML policy file, built-in profiles (claude-code/aider/cursor/codex/generic), append-only audit log with `aixgate audit tail`/`query`, `aixgate doctor` diagnostic |
| **v1.0** | future | Stable policy schema, signed audit log, FUSE-T fallback for macOS, comprehensive profile library |
| **v1.x** | future | Windows (WFP), eBPF backend on Linux, fleet management hook for `aixgo-cloud` |

Full roadmap and PRD: [docs/PRD.md](docs/PRD.md).

---

## Contributing

Aixgate accepts contributions under a Developer Certificate of Origin (DCO). Sign your commits with `git commit -s`.

Open issues for discussion before large changes. Good first contributions: better error messages around `sandbox-exec` exits, integration tests against more AI coding agents, documentation improvements.

---

## See also

- [aixgo](https://github.com/aixgo-dev/aixgo) — The Go framework for building AI agents that aixgate runs safely.
- [aixgo.dev](https://aixgo.dev) — Documentation and guides for the aixgo-dev family.

---

## License

[MIT](LICENSE).

> aixgo.dev builds agents. Aixgate keeps them in their lane.
