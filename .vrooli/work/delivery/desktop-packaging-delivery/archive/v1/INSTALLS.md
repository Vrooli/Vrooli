# Installs

External tools installed for this goal (`large-effort-orchestration` §5).

- Only the orchestrator authorizes an install, and records it here before or
  when it happens.
- Dependency installs go through Scenario Dependency Analyzer
  (`scenario-dependency-analyzer deps install …`).
- Host packages that need root go through a `internal/tools/<name>/tool.json`
  declaration and the operator's `sudo vrooli setup`
  (`docs/configuration/host/tools.md`). Agents never run `sudo`.
- Before the goal completes, stop everything it started outside the project
  and uninstall large goal-only installs.

Host state at commissioning (2026-10-06): Go 1.26.0, Node 20.19.4, pnpm,
xvfb-run present. Missing: `rustc`/`cargo`/`rustup`, `wails3`,
`WebKitWebDriver`, `tauri-driver`, `smem`, `vmtouch`, `hyperfine`, and the
WebKitGTK/GTK/libsoup development headers (`pkg-config` finds none). The
`libwebkit2gtk-4.1.so.0` runtime library is present.

| Tool | State | Size | Why | Route | Cleanup rule |
|---|---|---|---|---|---|
| Rust toolchain (`rustup`, stable `rustc`/`cargo`, user-level under `~/.rustup` and `~/.cargo`) | Pre-approved by the operator (DPF-FB-002); not installed | ~1.5 GB | Tauri 2 shell (E5) | User-level install, no root. Prefer a `internal/tools/rust/tool.json` declaration if E5 finds the tool registry fits; otherwise authorize and record here | Keep if `tauri-sidecar` stays a selectable strategy (it is a build-time dependency); otherwise remove at close |
| `tauri-cli` 2.x (`cargo install` or npm `@tauri-apps/cli` through SDA) | Pre-approved; not installed | ~50 MB | Build Tauri apps | npm package through SDA preferred | Same as Rust |
| Wails v3 CLI (`wails3`, `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.27`) | Pre-approved (beta) by the operator; not installed | ~40 MB | `wails-inprocess` strategy (E4) | Pin v3.0.0-beta.27 after official release review; route Go module changes through SDA | Keep if `wails-inprocess` stays selectable |
| WebKitGTK and GTK development headers (`libwebkit2gtk-4.1-dev`, `libgtk-3-dev`, `libsoup-3.0-dev`) | Declared in `internal/tools/webkitgtk/tool.json`; not installed; needs root | ~150 MB | Compile Wails on Linux | Required scenario host tool; operator runs `sudo vrooli setup` from QUEUE `## Needs operator` | Keep while any GTK3/WebKit2GTK strategy is selectable |
| `webkit2gtk-driver` (`WebKitWebDriver`) | Declared in `internal/tools/webkit2gtk-driver/tool.json`; not installed; needs root | ~1 MB | WebDriver automation adapter for WebKitGTK (E4/E5 parity) | Required scenario host tool; operator runs `sudo vrooli setup` from QUEUE `## Needs operator` | Keep while any WebKitGTK strategy is selectable |
| `vmtouch` or an in-repo `posix_fadvise` helper | Not installed | <1 MB | Cold-start page-cache eviction without root (E1) | A small Go helper in scenario-to-desktop (E1 Producer), not a host tool | n/a if implemented in repo |
