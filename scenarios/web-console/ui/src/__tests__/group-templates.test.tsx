import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent, waitFor } from "@testing-library/react";

import GroupModePanel from "../components/launcher/GroupModePanel";
import GroupTemplatesPanel from "../components/settings/GroupTemplatesPanel";
import type { GroupTemplateDTO } from "../api/grouptemplates";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";

// [REQ:P0-014g] Group Templates

const mockList = vi.fn();
const mockUpsert = vi.fn();
const mockDelete = vi.fn();

vi.mock("../api/grouptemplates", () => ({
  listGroupTemplates: (...a: unknown[]) => mockList(...a) as unknown,
  upsertGroupTemplate: (...a: unknown[]) => mockUpsert(...a) as unknown,
  deleteGroupTemplate: (...a: unknown[]) => mockDelete(...a) as unknown,
}));

// The shipped example is a TWO-role template; the fixture here is THREE, so
// the general case is what the tests exercise by default rather than the
// shape that happens to match one operator's workflow.
const threeRoles: GroupTemplateDTO = {
  id: "t1",
  name: "Plan, build, critique",
  color: "#22d3ee",
  use_count: 4,
  roles: [
    { label: "Planner", command: "claude", working_dir: "", incoming_prompt: "", backend: "", target_id: "", start_mode: "eager" },
    { label: "Implementer", command: "codex --yolo", working_dir: "", incoming_prompt: "Implement {{payload}}", backend: "", target_id: "", start_mode: "waiting" },
    { label: "Critic", command: "opencode", working_dir: "", incoming_prompt: "Critique {{payload}}", backend: "", target_id: "", start_mode: "waiting" },
  ],
};

/** The template control is a trigger and a menu, so choosing is two clicks. */
const chooseTemplate = (id: string) => {
  fireEvent.click(screen.getByTestId("launcher-template-picker"));
  fireEvent.click(screen.getByTestId(`launcher-template-option-${id}`));
};

describe("GroupModePanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockList.mockResolvedValue([threeRoles]);
    mockUpsert.mockResolvedValue(threeRoles);
    useWorkspaceStore.setState({ lastGroupTemplateId: null });
  });

  it("creates a group with no template at all", async () => {
    const onCreate = vi.fn();
    render(<GroupModePanel open onCreate={onCreate} isCreating={false} disabled={false} />);
    await waitFor(() => { expect(mockList).toHaveBeenCalled(); });

    fireEvent.change(screen.getByTestId("launcher-group-name"), { target: { value: "Refactor pass" } });
    fireEvent.click(screen.getByTestId("launcher-create-group"));

    expect(onCreate).toHaveBeenCalledWith(expect.objectContaining({ name: "Refactor pass", roles: [] }));
  });

  it("seeds the role list from a template, at its real length", async () => {
    render(<GroupModePanel open onCreate={vi.fn()} isCreating={false} disabled={false} />);
    await waitFor(() => { expect(mockList).toHaveBeenCalled(); });

    chooseTemplate("t1");
    await waitFor(() => { expect(screen.getByTestId("launcher-group-role-2")).toBeInTheDocument(); });
    // Three roles, not two: the model is a list, not a pair.
    expect(screen.queryByTestId("launcher-group-role-3")).not.toBeInTheDocument();
  });

  it("carries only the eager role as starting now", async () => {
    const onCreate = vi.fn();
    render(<GroupModePanel open onCreate={onCreate} isCreating={false} disabled={false} />);
    await waitFor(() => { expect(mockList).toHaveBeenCalled(); });

    chooseTemplate("t1");
    await waitFor(() => { expect(screen.getByTestId("launcher-group-role-0")).toBeInTheDocument(); });
    fireEvent.change(screen.getByTestId("launcher-group-name"), { target: { value: "Ship it" } });
    fireEvent.click(screen.getByTestId("launcher-create-group"));

    const request = onCreate.mock.calls[0]?.[0] as { roles: { start_mode: string }[] };
    expect(request.roles.filter((r) => r.start_mode === "eager")).toHaveLength(1);
    expect(request.roles).toHaveLength(3);
  });

  // The role list in the launcher is a working copy of the recipe.
  it("does not write launcher edits back to the stored template", async () => {
    const onCreate = vi.fn();
    render(<GroupModePanel open onCreate={onCreate} isCreating={false} disabled={false} />);
    await waitFor(() => { expect(mockList).toHaveBeenCalled(); });

    chooseTemplate("t1");
    await waitFor(() => { expect(screen.getByTestId("launcher-group-role-2")).toBeInTheDocument(); });
    fireEvent.click(screen.getByTestId("launcher-group-role-remove-2"));
    fireEvent.change(screen.getByTestId("launcher-group-name"), { target: { value: "Ship it" } });
    fireEvent.click(screen.getByTestId("launcher-create-group"));

    // Nothing was saved back.
    expect(mockUpsert).not.toHaveBeenCalled();
    const request = onCreate.mock.calls[0]?.[0] as { roles: unknown[] };
    expect(request.roles).toHaveLength(2);
  });

  it("adds a role by hand, with no template", async () => {
    const onCreate = vi.fn();
    render(<GroupModePanel open onCreate={onCreate} isCreating={false} disabled={false} />);
    await waitFor(() => { expect(mockList).toHaveBeenCalled(); });

    fireEvent.click(screen.getByTestId("launcher-group-role-add"));
    fireEvent.change(screen.getByTestId("launcher-group-role-label-0"), { target: { value: "Watcher" } });
    fireEvent.change(screen.getByTestId("launcher-group-role-command-0"), { target: { value: "tail -f log" } });
    fireEvent.change(screen.getByTestId("launcher-group-name"), { target: { value: "Debug" } });
    fireEvent.click(screen.getByTestId("launcher-create-group"));

    const request = onCreate.mock.calls[0]?.[0] as { roles: { label: string; start_mode: string }[] };
    expect(request.roles[0]).toMatchObject({ label: "Watcher", start_mode: "waiting" });
  });

  // Defaulting a hand-added role to eager would make adding a role quietly
  // cost a process.
  it("toggles a role between starting now and waiting", async () => {
    const onCreate = vi.fn();
    render(<GroupModePanel open onCreate={onCreate} isCreating={false} disabled={false} />);
    await waitFor(() => { expect(mockList).toHaveBeenCalled(); });

    fireEvent.click(screen.getByTestId("launcher-group-role-add"));
    fireEvent.click(screen.getByTestId("launcher-group-role-mode-0"));
    fireEvent.change(screen.getByTestId("launcher-group-name"), { target: { value: "Debug" } });
    fireEvent.click(screen.getByTestId("launcher-create-group"));

    const request = onCreate.mock.calls[0]?.[0] as { roles: { start_mode: string }[] };
    expect(request.roles[0]?.start_mode).toBe("eager");
  });

  // A waiting role exists to be handed something, so what it receives is on
  // the row rather than hidden in a template editor the operator must find.
  it("edits a waiting role's incoming message in place", async () => {
    const onCreate = vi.fn();
    render(<GroupModePanel open onCreate={onCreate} isCreating={false} disabled={false} />);
    await waitFor(() => { expect(mockList).toHaveBeenCalled(); });

    chooseTemplate("t1");
    await waitFor(() => { expect(screen.getByTestId("launcher-group-role-1")).toBeInTheDocument(); });

    fireEvent.change(screen.getByTestId("launcher-group-role-prompt-1"), { target: { value: "Ship {{payload}}" } });
    fireEvent.change(screen.getByTestId("launcher-group-name"), { target: { value: "Ship it" } });
    fireEvent.click(screen.getByTestId("launcher-create-group"));

    const request = onCreate.mock.calls[0]?.[0] as { roles: { incoming_prompt: string }[] };
    expect(request.roles[1]?.incoming_prompt).toBe("Ship {{payload}}");
  });

  // An eager role starts immediately, so there is nothing to hand it on
  // creation and no message field to fill in.
  it("offers the incoming message only while a role is waiting", async () => {
    render(<GroupModePanel open onCreate={vi.fn()} isCreating={false} disabled={false} />);
    await waitFor(() => { expect(mockList).toHaveBeenCalled(); });

    chooseTemplate("t1");
    await waitFor(() => { expect(screen.getByTestId("launcher-group-role-0")).toBeInTheDocument(); });

    expect(screen.queryByTestId("launcher-group-role-prompt-0")).toBeNull();
    fireEvent.click(screen.getByTestId("launcher-group-role-mode-0"));
    expect(screen.getByTestId("launcher-group-role-prompt-0")).toBeInTheDocument();
  });

  it("says the list is empty rather than showing nothing at all", async () => {
    render(<GroupModePanel open onCreate={vi.fn()} isCreating={false} disabled={false} />);
    await waitFor(() => { expect(mockList).toHaveBeenCalled(); });
    expect(screen.getByTestId("launcher-group-role-empty")).toBeInTheDocument();
  });

  it("hands the machine control a place beside the template trigger", async () => {
    render(
      <GroupModePanel
        open
        onCreate={vi.fn()}
        isCreating={false}
        disabled={false}
        machineSlot={<div data-testid="test-machine-slot" />}
      />,
    );
    await waitFor(() => { expect(mockList).toHaveBeenCalled(); });
    expect(screen.getByTestId("test-machine-slot")).toBeInTheDocument();
  });

  it("refuses to create a group with no name", async () => {
    render(<GroupModePanel open onCreate={vi.fn()} isCreating={false} disabled={false} />);
    await waitFor(() => { expect(mockList).toHaveBeenCalled(); });
    expect(screen.getByTestId("launcher-create-group")).toBeDisabled();
  });
});

  // Most of the time the template you want is the one you used last, so the
  // panel offers it rather than making you pick again.
  it("preselects the template chosen last", async () => {
    useWorkspaceStore.setState({ lastGroupTemplateId: "t1" });
    render(<GroupModePanel open onCreate={vi.fn()} isCreating={false} disabled={false} />);
    await waitFor(() => { expect(screen.getByTestId("launcher-group-role-2")).toBeInTheDocument(); });
    expect(screen.getByTestId("launcher-template-picker")).toHaveTextContent(threeRoles.name);
  });

  // A remembered id that no longer resolves must not preselect a name that
  // points at nothing.
  it("ignores a remembered template that has since been deleted", async () => {
    useWorkspaceStore.setState({ lastGroupTemplateId: "deleted" });
    render(<GroupModePanel open onCreate={vi.fn()} isCreating={false} disabled={false} />);
    await waitFor(() => { expect(mockList).toHaveBeenCalled(); });
    expect(screen.getByTestId("launcher-group-role-empty")).toBeInTheDocument();
  });

  it("remembers each choice, including going back to no template", async () => {
    render(<GroupModePanel open onCreate={vi.fn()} isCreating={false} disabled={false} />);
    await waitFor(() => { expect(mockList).toHaveBeenCalled(); });

    chooseTemplate("t1");
    expect(useWorkspaceStore.getState().lastGroupTemplateId).toBe("t1");

    // "No template" is a choice too, so it replaces the memory rather than
    // leaving the previous pick to reassert itself on the next open.
    chooseTemplate("none");
    expect(useWorkspaceStore.getState().lastGroupTemplateId).toBeNull();
  });

describe("GroupTemplatesPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockList.mockResolvedValue([threeRoles]);
    mockUpsert.mockResolvedValue(threeRoles);
    mockDelete.mockResolvedValue(undefined);
  });

  it("lists saved templates", async () => {
    render(<GroupTemplatesPanel />);
    await waitFor(() => { expect(screen.getByTestId("group-template-t1")).toBeInTheDocument(); });
  });

  // Prohibition 5, as a test: the delete control on a shipped example is the
  // same control as on one the operator wrote, with no guard.
  it("deletes any template, including a shipped example", async () => {
    mockList.mockResolvedValue([{ ...threeRoles, id: "example-plan-then-implement", name: "Plan → Implement" }]);
    render(<GroupTemplatesPanel />);
    await waitFor(() => { expect(screen.getByTestId("group-template-example-plan-then-implement")).toBeInTheDocument(); });

    mockList.mockResolvedValue([]);
    fireEvent.click(screen.getByTestId("group-template-delete-example-plan-then-implement"));

    await waitFor(() => { expect(mockDelete).toHaveBeenCalledWith("example-plan-then-implement"); });
    // With the example gone the panel still works and offers creation.
    await waitFor(() => { expect(screen.getByTestId("group-templates-empty")).toBeInTheDocument(); });
    expect(screen.getByTestId("group-templates-create")).toBeInTheDocument();
  });

  it("creates a template with a role list of any length", async () => {
    render(<GroupTemplatesPanel />);
    await waitFor(() => { expect(mockList).toHaveBeenCalled(); });

    fireEvent.click(screen.getByTestId("group-templates-create"));
    fireEvent.change(screen.getByTestId("group-template-name"), { target: { value: "Four ways" } });
    fireEvent.click(screen.getByTestId("group-templates-role-add"));
    fireEvent.click(screen.getByTestId("group-templates-role-add"));
    fireEvent.click(screen.getByTestId("group-template-save"));

    await waitFor(() => { expect(mockUpsert).toHaveBeenCalled(); });
    const saved = mockUpsert.mock.calls[0]?.[0] as { name: string; roles: unknown[] };
    expect(saved.name).toBe("Four ways");
    expect(saved.roles).toHaveLength(3);
  });

  // Editing content must not reset how often a template has been used.
  it("omits use_count when saving a content edit", async () => {
    render(<GroupTemplatesPanel />);
    await waitFor(() => { expect(mockList).toHaveBeenCalled(); });

    fireEvent.click(screen.getByTestId("group-templates-create"));
    fireEvent.change(screen.getByTestId("group-template-name"), { target: { value: "Edited" } });
    fireEvent.click(screen.getByTestId("group-template-save"));

    await waitFor(() => { expect(mockUpsert).toHaveBeenCalled(); });
    expect(mockUpsert.mock.calls[0]?.[0]).not.toHaveProperty("use_count");
  });
});
