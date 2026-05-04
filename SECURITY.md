# Security Policy

Aixgate is a security tool. We take vulnerability reports seriously and respond quickly.

## Reporting a Vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Instead, use [GitHub Security Advisories](https://github.com/aixgo-dev/aixgate/security/advisories/new) for private disclosure. We will acknowledge receipt within **2 business days** and aim for an initial assessment within **5 business days**.

If you cannot use GitHub Security Advisories, contact the maintainers directly via the email listed in the repository's homepage profile or in [CODEOWNERS](.github/CODEOWNERS) (when present).

## Scope

In scope:

- Sandbox bypass: any technique that allows a process running under `aixgate run` to read, write, or execute outside the configured policy without exiting the sandbox.
- Policy parser vulnerabilities: malformed `.aixgate.yaml` files that crash the policy compiler, leak filesystem information, or allow privilege escalation when loaded.
- Audit log integrity: any technique that allows a sandboxed process to suppress, forge, or tamper with audit log entries.
- Privilege escalation from the sandboxed process to the host user.
- Information disclosure beyond what the policy allows (e.g. file existence leaks via `ENOENT` vs `EACCES` distinction on platforms where Aixgate guarantees `ENOENT`).

Out of scope:

- Attacks that require an already-rooted host. Aixgate does not defend against root-level attackers (PRD §3.3 anti-goal).
- Vulnerabilities in the AI coding agent being sandboxed (Claude Code, Cursor, Aider, etc.) — report those upstream.
- macOS path-existence visibility under `sandbox-exec` (documented limitation, PRD §15.2 Milestone 2): `sandbox-exec` returns `EACCES` rather than hiding existence; this is a known platform asymmetry, not a vulnerability.
- Kernel-level escapes from Linux Landlock or seccomp — those are kernel CVEs, report to the kernel security team.
- DoS via legitimate policy primitives (e.g. policies that deny everything are not a vulnerability).

## Embargo and Coordinated Disclosure

For high-severity findings, we follow a coordinated disclosure timeline:

1. **Acknowledgement** within 2 business days.
2. **Initial assessment** within 5 business days, including severity rating and fix ETA.
3. **Fix development** in a private branch.
4. **Coordinated release** with the reporter — typically 30–90 days from the initial report depending on severity, with extensions possible if the fix requires upstream changes (e.g. to a Go-FUSE or Landlock dependency).
5. **Public advisory** at release time, crediting the reporter unless they request anonymity.

Aixgate maintains its own CVE namespace and embargo process, distinct from the [aixgo](https://github.com/aixgo-dev/aixgo) framework's. A vulnerability in `aixgo/pkg/security` that Aixgate consumes will be coordinated jointly.

## Supported Versions

During pre-v1.0 development, only the latest minor release receives security fixes. After v1.0, the policy will be updated to clarify long-term support windows.

## See Also

- [PRD §3.3 — Threat Model](docs/PRD.md) for what Aixgate is and is not designed to defend against.
- [aixgo SECURITY.md](https://github.com/aixgo-dev/aixgo/blob/main/SECURITY.md) (when present) for the framework's policy.
