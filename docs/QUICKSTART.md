# Vrooli Quick Start

This is the canonical first-touch guide for the project-level Vrooli platform.

## What You Are Starting

Vrooli is a local, cross-platform platform for orchestrating:

- resources such as databases, inference services, automation systems, and supporting infrastructure
- scenarios that compose those capabilities into products, internal tools, operator surfaces, and meta-systems

The root control surface is the Go-native `vrooli` CLI.

## Prerequisites

The fresh-machine bootstrap needs only a POSIX shell and `curl`. If the OS does
not already ship `tar` or OpenSSL, the installer obtains them through the native
package manager before authenticating the release. Go, git, Homebrew, Node,
pnpm, and Docker are **not** manual bootstrap prerequisites.

- **Linux**: setup uses the available apt/dnf/yum/pacman/apk package manager.
- **macOS (Apple Silicon or Intel)**: setup bootstraps Homebrew when absent, then
  installs git, Go, and the selected host tools through it.
- **Windows**: use WSL2 with a Linux distribution and follow the POSIX flow.
- **Docker**: required lazily only when an enabled resource declares a
  Docker-service or Compose runtime; `--resources none` does not demand it.

macOS caveats: Workspace Sandbox uses partial Seatbelt containment (filesystem-write and network denial; no Linux namespaces or `/workspace` path illusion), while X11 desktop automation tools remain unavailable and skip cleanly. Containerized Ollama runs CPU-only (no Metal passthrough), and some resources remain Linux-only (check `resource.json`). Background supervisor persistence uses a launchd LaunchAgent (`vrooli runtime supervisor install --user`) instead of systemd. See the canonical [platform support matrix](reference/platform-support.md) for evidence tiers and unqualified hardware capabilities.

## Setup

```bash
curl -fsSL https://raw.githubusercontent.com/Vrooli/Vrooli/main/install/install.sh | sh
export PATH="$HOME/.vrooli/bin:$PATH"
vrooli setup
```

The signed installer downloads the matching prebuilt CLI and source archive.
For an existing checkout, `make setup` remains a convenience wrapper and uses
the installed CLI when present.

## Start The Development Stack

```bash
vrooli develop
```

For project-level command discovery:

```bash
vrooli help
```

## Inspect The Platform

```bash
vrooli status
vrooli scenario list
vrooli resource list
```

## Work With A Scenario

Scenario-local Makefiles are the preferred operational path for individual scenarios:

```bash
cd scenarios/<scenario-name>
make start
make test
make logs
make stop
```

The root CLI remains available when you need cross-scenario operations:

```bash
vrooli scenario list
vrooli scenario info <name>
vrooli scenario start <name>
vrooli scenario test <name>
vrooli scenario logs <name>
```

## Work With Resources

```bash
vrooli resource list
vrooli resource status
vrooli resource start-all
vrooli resource start postgres
vrooli resource logs postgres
```

## Testing

Run project-level validation with:

```bash
make test
```

Run scenario-focused testing with:

```bash
vrooli scenario test <name>
```

For the deeper testing workflow, see [TESTING.md](TESTING.md).

## Where To Go Next

- [README.md](README.md) for the docs hub
- [concepts/ARCHITECTURE.md](concepts/ARCHITECTURE.md) for the platform mental model
- [concepts/GLOSSARY.md](concepts/GLOSSARY.md) for core terminology
- [reference/cli-commands.md](reference/cli-commands.md) for command reference
- [deployment/README.md](deployment/README.md) for deployment tiers and current packaging reality
