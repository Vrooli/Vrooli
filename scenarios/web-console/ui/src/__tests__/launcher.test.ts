import { describe, it, expect, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { renderWithProviders as render } from "../test-utils";
import TerminalLauncher from "../components/TerminalLauncher";
import { createElement } from "react";
import { shortcutsClient } from "../api/shortcuts";

// [REQ:P0-006a] Terminal Launch Flow UI
// [REQ:P0-006b] Configurable Shortcut Entries
describe("TerminalLauncher", () => {
  it("component module exports default function", async () => {
    const mod = await import("../components/TerminalLauncher");
    expect(typeof mod.default).toBe("function");
  });

  it("ShortcutEntry interface includes label, command, and description", async () => {
    const entry: import("../consts/shortcuts").ShortcutEntry = {
      label: "Test",
      command: "echo hello",
      description: "A test shortcut",
    };
    expect(entry.label).toBe("Test");
    expect(entry.command).toBe("echo hello");
    expect(entry.description).toBe("A test shortcut");
  });

  it("renders local and remote target states, options, search, and launch actions", async () => {
    const onLaunch = vi.fn();
    const onClose = vi.fn();
    const onRefreshTargets = vi.fn();
    const target = (id: string, state: "dispatchable" | "offline" | "needs-update" | "unavailable", available: boolean) => ({
      id, kind: "bridge-node" as const, label: `Node ${id}`, os: "linux", arch: "amd64", node_id: id,
      available, state, recovery_action: available ? undefined : "repair target", last_seen_at: id === "bad-date" ? "not-a-date" : undefined,
      readiness: [{ key: "bridge", label: "Bridge", passed: available, detail: "status" }],
    });
    render(createElement(TerminalLauncher, {
      open: true,
      onClose,
      onLaunch,
      onRefreshTargets,
      shortcuts: [{ label: "List files", command: "ls", description: "show files" }, { label: "Codex", command: "codex login --device-auth" }],
      availableBackends: [{ id: "standard", display_name: "Standard", description: "", available: true, survives_restart: false }, { id: "persistent", display_name: "Persistent", description: "", available: true, survives_restart: true }],
      targetCatalog: { status: "configured-empty", targets: [target("offline", "offline", false), target("update", "needs-update", false), target("ready", "dispatchable", true), target("bad-date", "unavailable", false)], message: "catalog warning", recovery_action: "refresh" },
    }));

    expect(screen.getByTestId("terminal-launcher")).toBeInTheDocument();
    expect(screen.getByTestId("launcher-target-card-local")).toBeInTheDocument();
    expect(screen.getByTestId("launcher-target-search")).toBeInTheDocument();
    fireEvent.change(screen.getByTestId("launcher-target-search"), { target: { value: "ready" } });
    expect(screen.getByTestId("launcher-target-card-ready")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("launcher-target-card-ready"));
    fireEvent.keyDown(screen.getByTestId("launcher-target-card-ready"), { key: "Home" });
    fireEvent.keyDown(screen.getByTestId("launcher-target-card-local"), { key: "End" });
    fireEvent.click(screen.getByTestId("launcher-options-toggle"));
    fireEvent.change(screen.getByTestId("launcher-working-dir"), { target: { value: " /tmp " } });
    fireEvent.change(screen.getByTestId("launcher-backend-select"), { target: { value: "persistent" } });
    fireEvent.change(screen.getByTestId("launcher-timeout-select"), { target: { value: "preset:1h" } });
    fireEvent.click(screen.getByTestId("launcher-empty-shell"));
    expect(onLaunch).toHaveBeenCalledWith(expect.objectContaining({ backend: "persistent", workingDir: "/tmp" }));
    fireEvent.change(screen.getByTestId("launcher-custom-input"), { target: { value: "echo hi" } });
    fireEvent.keyDown(screen.getByTestId("launcher-custom-input"), { key: "Enter" });
    fireEvent.click(screen.getByTestId("launcher-shortcut-list-files"));
    await waitFor(() => expect(onRefreshTargets).not.toHaveBeenCalled());
    fireEvent.click(screen.getByTestId("launcher-target-refresh"));
    expect(onRefreshTargets).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByLabelText(/close/i));
    expect(onClose).toHaveBeenCalled();
  });

  it("shows the unavailable and creating states", () => {
    render(createElement(TerminalLauncher, { open: true, onClose: vi.fn(), onLaunch: vi.fn(), isCreating: true, availableTargets: [] }));
    expect(screen.getByTestId("launcher-creating")).toBeInTheDocument();
    expect(screen.getByTestId("launcher-custom-launch")).toBeDisabled();
  });

  it("loads effective shortcuts, renders catalog errors, and handles unavailable targets", async () => {
    const getEffective = vi.spyOn(shortcutsClient, "getEffective").mockResolvedValue({
      shortcuts: [{ label: "Status", command: "status", description: "" }],
    } as never);
    const onLaunch = vi.fn();
    const remote = {
      id: "offline-node",
      kind: "bridge-node" as const,
      label: "Offline node",
      available: false,
      state: "offline" as const,
      failure_rung: "node unreachable",
      last_seen_at: "not-a-date",
      readiness: [{ key: "bridge", label: "Bridge", passed: false, detail: "down" }],
    };
    render(createElement(TerminalLauncher, {
      open: true,
      onClose: vi.fn(),
      onLaunch,
      availableBackends: [],
      availableTargets: [remote],
      targetCatalog: { status: "registry-error", targets: [remote], message: "registry offline", recovery_action: "retry" },
    }));

    expect(screen.getByTestId("launcher-no-backend")).toHaveTextContent("No terminal backend");
    expect(screen.getByTestId("launcher-target-catalog-state")).toHaveTextContent("terminalLauncher.registryError");
    fireEvent.click(screen.getByTestId("launcher-target-card-offline-node"));
    expect(screen.getByText("terminalLauncher.targetUnavailable")).toBeInTheDocument();
    expect(screen.getByTestId("launcher-empty-shell")).toBeDisabled();
    expect(screen.getByTestId("launcher-custom-launch")).toBeDisabled();
    await waitFor(() => expect(screen.getByTestId("launcher-shortcut-status")).toBeInTheDocument());
    expect(getEffective).toHaveBeenCalled();
    getEffective.mockRestore();
  });

  it("shows loading and empty remote-search states, with keyboard navigation", () => {
    const makeRemote = (id: string) => ({
      id,
      kind: "bridge-node" as const,
      label: `Node ${id}`,
      os: "linux",
      arch: "amd64",
      available: true,
      state: "dispatchable" as const,
    });
    const remotes = ["one", "two", "three", "four"].map(makeRemote);
    const { rerender } = render(createElement(TerminalLauncher, {
      open: true,
      onClose: vi.fn(),
      onLaunch: vi.fn(),
      targetsLoading: true,
      targetCatalog: { status: "unconfigured", targets: [] },
    }));
    expect(screen.getByTestId("launcher-target-loading")).toBeInTheDocument();
    expect(screen.getByTestId("launcher-target-catalog-state")).toBeInTheDocument();

    rerender(createElement(TerminalLauncher, {
      open: true,
      onClose: vi.fn(),
      onLaunch: vi.fn(),
      defaultPolicy: { mode: "preset", duration: "1h" },
      targetCatalog: { status: "ready", targets: remotes },
    }));
    const search = screen.getByTestId("launcher-target-search");
    fireEvent.change(search, { target: { value: "does-not-exist" } });
    expect(screen.getByText("terminalLauncher.noRemoteNodes")).toBeInTheDocument();
    fireEvent.keyDown(screen.getByTestId("launcher-target-card-local"), { key: "ArrowDown" });
    fireEvent.keyDown(screen.getByTestId("launcher-target-card-local"), { key: "ArrowLeft" });
    fireEvent.keyDown(screen.getByTestId("launcher-target-card-local"), { key: "Escape" });
  });

  it("handles closed launchers, custom backend reasons, and remote last-seen fallbacks", async () => {
    const getEffective = vi.spyOn(shortcutsClient, "getEffective").mockRejectedValue(new Error("offline"));
    const remote = {
      id: "remote-node",
      kind: "bridge-node" as const,
      label: "Remote node",
      available: true,
      last_seen_at: "not-a-date",
    };
    const { rerender } = render(createElement(TerminalLauncher, {
      open: false,
      onClose: vi.fn(),
      onLaunch: vi.fn(),
      availableBackends: [{ id: "standard", display_name: "Standard", description: "", available: false, survives_restart: false, reason: "backend registry offline" }],
      availableTargets: [remote],
      onRefreshTargets: vi.fn(),
    }));
    expect(getEffective).not.toHaveBeenCalled();

    rerender(createElement(TerminalLauncher, {
      open: true,
      onClose: vi.fn(),
      onLaunch: vi.fn(),
      availableBackends: [{ id: "standard", display_name: "Standard", description: "", available: false, survives_restart: false, reason: "backend registry offline" }],
      availableTargets: [remote],
      onRefreshTargets: vi.fn(),
      targetsLoading: true,
    }));
    expect(screen.getByTestId("launcher-no-backend")).toHaveTextContent("backend registry offline");
    expect(screen.getByTestId("launcher-target-refresh")).toBeDisabled();

    rerender(createElement(TerminalLauncher, {
      open: true,
      onClose: vi.fn(),
      onLaunch: vi.fn(),
      availableBackends: [{ id: "standard", display_name: "Standard", description: "", available: false, survives_restart: false, reason: "backend registry offline" }],
      availableTargets: [remote],
      onRefreshTargets: vi.fn(),
    }));
    fireEvent.click(screen.getByTestId("launcher-target-card-remote-node"));
    await waitFor(() => expect(screen.getByTestId("launcher-target-card-remote-node")).toHaveAttribute("aria-selected", "true"));
    getEffective.mockRestore();
  });
});
