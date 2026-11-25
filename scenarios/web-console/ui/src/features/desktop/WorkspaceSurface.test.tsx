import { fireEvent, screen } from "@testing-library/react";
import { renderWithProviders as render } from "../../test-utils";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { WorkspacePaneStateStatus, WorkspacePaneToolbar, WorkspaceSurfacePane, projectDesktopState, projectFilePreviewState, projectRemoteSurfaceState } from "./WorkspaceSurface";
import { workspaceSurfacePresentationAttributes } from "./workspaceSurfaceContract";

describe("shared file/browser/device pane contract", () => {
  it("projects owner-specific states into one vocabulary", () => {
    expect(projectRemoteSurfaceState("viewer")).toBe("observe");
    expect(projectRemoteSurfaceState("controller")).toBe("controlling");
    expect(projectRemoteSurfaceState("blocked", false)).toBe("permission-blocked");
    expect(projectRemoteSurfaceState("offline")).toBe("offline");
    expect(projectRemoteSurfaceState("revoked")).toBe("revoked");
    expect(projectRemoteSurfaceState("disconnected")).toBe("reconnecting");
    expect(projectDesktopState("viewer", true)).toBe("observe");
    expect(projectDesktopState("revoked", true)).toBe("revoked");
    expect(projectDesktopState("offline", false)).toBe("offline");
    expect(projectDesktopState("connecting", false)).toBe("loading");
    expect(projectFilePreviewState(true, "resolving")).toBe("loading");
    expect(projectFilePreviewState(true, "ready")).toBe("ready");
    expect(projectFilePreviewState(true, "choosing")).toBe("choosing");
    expect(projectFilePreviewState(true, "unsupported")).toBe("unsupported");
    expect(projectFilePreviewState(false, "idle")).toBe("closed");
  });

  it.each([
    ["loading", "loading"], ["ready", "ready"], ["observe", "ready"], ["controlling", "ready"],
    ["permission-blocked", "blocked"], ["offline", "offline"], ["revoked", "revoked"],
    ["choosing", "choosing"], ["unsupported", "unsupported"], ["reconnecting", "reconnecting"],
    ["failed", "failed"], ["closed", "closed"],
  ] as const)("projects %s into the shared surface lifecycle", (state, lifecycle) => {
    const attributes = workspaceSurfacePresentationAttributes("bas", state, false, true);
    expect(attributes["data-surface-lifecycle"]).toBe(lifecycle);
    expect(attributes["data-surface-access"]).toBe(state === "observe" ? "observe" : state === "controlling" ? "control" : "read-only");
  });

  it.each([
    ["permission-blocked", /Access is blocked by this surface's permission/],
    ["offline", /surface is offline/i],
    ["choosing", /waiting for a selection/i],
    ["unsupported", /format is not supported/i],
  ] as const)("announces %s with a specific status", (state, message) => {
    render(<WorkspacePaneStateStatus state={state} label="File preview" />);
    expect(screen.getByRole("status")).toHaveTextContent(message);
  });

  it.each([375, 1280])("shares status and keyboard behavior at %i px", async (width) => {
    const user = userEvent.setup();
    render(<div style={{ width }}><WorkspaceSurfacePane label="Browser session" title="Browser" owner="bas" canControl state="observe">
      <WorkspacePaneToolbar label="Browser session controls" state="observe">
        <button type="button">First</button><button type="button">Second</button>
      </WorkspacePaneToolbar>
    </WorkspaceSurfacePane><button type="button">After toolbar</button></div>);

    const pane = screen.getByTestId("workspace-surface-pane");
    expect(pane).toHaveAttribute("data-state", "observe");
    expect(pane).toHaveAttribute("data-surface-owner", "bas");
    expect(pane).toHaveAttribute("data-surface-access", "observe");
    expect(pane).toHaveAttribute("data-surface-can-control", "true");
    expect(screen.getByRole("status")).toHaveTextContent(/Observe mode/);
    const toolbar = screen.getByRole("toolbar", { name: "Browser session controls" });
    const [first, second] = screen.getAllByRole("button");
    expect(first).toHaveAttribute("tabindex", "0");
    expect(second).toHaveAttribute("tabindex", "-1");
    first?.focus();
    expect(pane).toHaveAttribute("data-surface-focused", "true");
    fireEvent.keyDown(first!, { key: "ArrowRight" });
    expect(second).toHaveFocus();
    expect(first).toHaveAttribute("tabindex", "-1");
    expect(second).toHaveAttribute("tabindex", "0");
    fireEvent.keyDown(second!, { key: "End" });
    expect(second).toHaveFocus();
    fireEvent.keyDown(second!, { key: "Home" });
    expect(first).toHaveFocus();
    const tab = new KeyboardEvent("keydown", { key: "Tab", bubbles: true, cancelable: true });
    first?.dispatchEvent(tab);
    expect(tab.defaultPrevented).toBe(false);
    first?.focus();
    await user.tab();
    expect(screen.getByRole("button", { name: "After toolbar" })).toHaveFocus();
    expect(pane).toHaveAttribute("data-surface-focused", "false");
    expect(toolbar).toHaveAttribute("aria-orientation", "horizontal");
  });
});
