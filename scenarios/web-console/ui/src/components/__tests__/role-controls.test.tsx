import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders as render } from "../../test-utils";
import RoleMenu from "../RoleMenu";
import RoleRow from "../RoleRow";
import type { RoleMeta, TabGroupMeta } from "../../stores/useWorkspaceStore";

const role: RoleMeta = { id: "role-1", groupId: "group-1", label: "Implementer", command: "codex", workingDir: "/tmp", incomingPrompt: "", backend: "", targetId: "", sessionId: null, sortOrder: 0 };
const group: TabGroupMeta = { id: "group-1", name: "Build", color: "#0ff", isCollapsed: false };

describe("waiting role controls", () => {
  it("starts tab and sidebar roles, hands off, and opens the menu at its button", () => {
    const onStart = vi.fn();
    const onHandoff = vi.fn();
    const onOpenMenu = vi.fn();
    const view = render(<RoleRow role={role} group={group} onStart={onStart} onHandoff={onHandoff} onOpenMenu={onOpenMenu} isLastInGroup />);
    fireEvent.click(screen.getByTestId("sidebar-waiting-role-start-role-1"));
    fireEvent.click(screen.getByTestId("sidebar-waiting-role-handoff-role-1"));
    fireEvent.click(screen.getByTestId("sidebar-waiting-role-menu-role-1"));
    expect(onStart).toHaveBeenCalledWith(role);
    expect(onHandoff).toHaveBeenCalledWith(role);
    expect(onOpenMenu).toHaveBeenCalledWith(role, expect.objectContaining({ x: expect.any(Number), y: expect.any(Number) }));

    view.unmount();
    render(<RoleRow role={role} group={group} onStart={onStart} onOpenMenu={onOpenMenu} isLastInGroup variant="tab" />);
    fireEvent.click(screen.getByTestId("tab-waiting-role-role-1"));
    expect(onStart).toHaveBeenCalledTimes(2);
  });

  it("routes each role menu action and dismisses when the library closes it", () => {
    const onStart = vi.fn();
    const onRename = vi.fn();
    const onEditPrompt = vi.fn();
    const onDelete = vi.fn();
    const onDismiss = vi.fn();
    render(<RoleMenu role={role} position={{ x: 10, y: 20 }} onStart={onStart} onRename={onRename} onEditPrompt={onEditPrompt} onDelete={onDelete} onDismiss={onDismiss} />);
    fireEvent.click(screen.getByTestId("role-menu-start"));
    fireEvent.click(screen.getByTestId("role-menu-rename"));
    fireEvent.click(screen.getByTestId("role-menu-edit-prompt"));
    fireEvent.click(screen.getByTestId("role-menu-delete"));
    expect(onStart).toHaveBeenCalledWith(role);
    expect(onRename).toHaveBeenCalledWith(role);
    expect(onEditPrompt).toHaveBeenCalledWith(role);
    expect(onDelete).toHaveBeenCalledWith(role);
  });
});
