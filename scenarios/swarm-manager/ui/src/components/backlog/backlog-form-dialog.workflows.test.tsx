import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen } from "@testing-library/react";
import { getSpatialNav } from "@vrooli/iframe-bridge/spatial";
import { renderWithProviders } from "../../test-utils";
import { defaultApiClient } from "../../lib/api-client";
import { useBacklogFormStore } from "../../stores";
import type { BacklogFormValues } from "../../types";
import { BacklogFormDialog } from "./backlog-form-dialog";

const original: BacklogFormValues = { name: "owner-work", title: "Owner work", description: "Owner description", status: "ready", priority: 4, tags: ["owner"], kind: "fix", dependsOn: ["idea/owner-dependency"], milestone: "owner-milestone", effort: "S", acceptanceAllow: ["src/**"], acceptanceDeny: ["*.lock"], executionMode: "sliced", continuation: "manual", scopePolicy: "fixed" };
function change(label: string, value: string) { fireEvent.change(screen.getByLabelText(label), { target: { value } }); }
beforeEach(() => {
  useBacklogFormStore.getState().reset();
  vi.spyOn(defaultApiClient, "get").mockImplementation(async path => { if (path === "/execution/strategies") return { items: [{ id: "sliced", display_name: "Sliced" }, { id: "goal", display_name: "Goal" }] }; throw new Error(`Unexpected form read ${path}`); });
  for (const method of ["post", "put", "patch", "delete"] as const) vi.spyOn(defaultApiClient, method).mockRejectedValue(new Error(`Unexpected form mutation ${method}`));
  vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("Unexpected form fetch")));
});
afterEach(() => { cleanup(); getSpatialNav()?.dispose(); for (const [path] of vi.mocked(defaultApiClient.get).mock.calls) expect(path).toBe("/execution/strategies"); for (const method of ["post", "put", "patch", "delete"] as const) expect(defaultApiClient[method]).not.toHaveBeenCalled(); expect(fetch).not.toHaveBeenCalled(); vi.restoreAllMocks(); vi.unstubAllGlobals(); useBacklogFormStore.getState().reset(); });

describe("Backlog form callback workflows", () => {
  it("rejects empty title and non-folder-safe identity before parent submission", () => {
    const submit = vi.fn(); renderWithProviders(<BacklogFormDialog isOpen mode="create" onSubmit={submit} onClose={vi.fn()} />);
    fireEvent.click(screen.getByRole("button", { name: "Create Idea" })); expect(screen.getByText("Title is required.")).toBeVisible(); expect(submit).not.toHaveBeenCalled();
    change("Title", "!!!"); fireEvent.click(screen.getByRole("button", { name: "Create Idea" })); expect(screen.getByText("Name is required.")).toBeVisible(); expect(submit).not.toHaveBeenCalled();
  });
  it("generates identity from a new title until the operator explicitly edits it", () => {
    const submit = vi.fn(); renderWithProviders(<BacklogFormDialog isOpen mode="create" onSubmit={submit} onClose={vi.fn()} />);
    change("Title", "First useful title"); expect(screen.getByLabelText("Name")).toHaveValue("first-useful-title");
    change("Name", "explicit-owner-name"); change("Title", " Second useful title "); expect(screen.getByLabelText("Name")).toHaveValue("explicit-owner-name");
    fireEvent.click(screen.getByRole("button", { name: "Create Idea" })); expect(submit.mock.calls[0]?.[0]).toEqual({ name: "explicit-owner-name", title: "Second useful title", description: "", status: "backlog", priority: 5, tags: [], kind: "idea", dependsOn: undefined, milestone: undefined, effort: undefined, acceptanceAllow: undefined, acceptanceDeny: undefined, executionMode: "sliced", executionLimits: undefined, continuation: "manual", scopePolicy: "fixed" });
  });
  it("submits trimmed current dependencies, acceptance patterns and optional fields without a network mutation", () => {
    const submit = vi.fn(); renderWithProviders(<BacklogFormDialog isOpen mode="create" defaultKind="research" onSubmit={submit} onClose={vi.fn()} />);
    change("Title", " Research bounded source "); change("Name", "research-bounded-source"); change("Description", " Read-only investigation "); change("Tags", "review, evidence"); change("Dependencies", " idea/source, ,fix/owner , "); change("Milestone", " source-milestone "); change("Effort", "M"); change("Acceptance Allow", " docs/**, ,src/** "); change("Acceptance Deny", " *.lock, node_modules/** ");
    fireEvent.click(screen.getByRole("button", { name: "Create Research" })); expect(submit.mock.calls).toEqual([[{ name: "research-bounded-source", title: "Research bounded source", description: "Read-only investigation", status: "backlog", priority: 5, tags: ["review", "evidence"], kind: "research", dependsOn: ["idea/source", "fix/owner"], milestone: "source-milestone", effort: "M", acceptanceAllow: ["docs/**", "src/**"], acceptanceDeny: ["*.lock", "node_modules/**"], executionMode: "sliced", executionLimits: undefined, continuation: "manual", scopePolicy: "fixed" }]]);
  });
  it("keeps edit identity fixed and returns the explicit current lifecycle status", () => {
    const submit = vi.fn(); renderWithProviders(<BacklogFormDialog isOpen mode="edit" initialValues={original} onSubmit={submit} onClose={vi.fn()} />);
    expect(screen.getByLabelText("Name")).toBeDisabled(); expect(screen.getByLabelText("Name")).toHaveValue("owner-work"); expect(screen.queryByLabelText("Backlog type")).toBeNull();
    change("Title", " Edited owner work "); change("Status", "completed"); fireEvent.click(screen.getByRole("button", { name: "Save Changes" })); expect(submit.mock.calls).toEqual([[{ ...original, title: "Edited owner work", status: "completed", executionLimits: undefined }]]);
  });
  it("retains a parent-refused draft and does not submit until another explicit action", () => {
    const submit = vi.fn(), close = vi.fn(); const view = renderWithProviders(<BacklogFormDialog isOpen mode="edit" initialValues={original} onSubmit={submit} onClose={close} />);
    change("Title", "Unaccepted edit"); view.rerender(<BacklogFormDialog isOpen mode="edit" initialValues={original} submitError="Owner refused edit" onSubmit={submit} onClose={close} />);
    expect(screen.getByText("Owner refused edit")).toBeVisible(); expect(screen.getByLabelText("Title")).toHaveValue("Unaccepted edit"); expect(submit).not.toHaveBeenCalled(); expect(close).not.toHaveBeenCalled();
  });
  it("honors controlled submission state across all real fields and explicit buttons", () => {
    const submit = vi.fn(), close = vi.fn(); const view = renderWithProviders(<BacklogFormDialog isOpen mode="edit" initialValues={original} onSubmit={submit} onClose={close} />); change("Title", "Pending draft");
    view.rerender(<BacklogFormDialog isOpen mode="edit" initialValues={original} isSubmitting onSubmit={submit} onClose={close} />);
    for (const field of [...screen.getAllByRole("textbox"), ...screen.getAllByRole("combobox"), ...screen.getAllByRole("spinbutton"), ...screen.getAllByRole("radio")]) expect(field).toBeDisabled();
    const save = screen.getByRole("button", { name: "Saving..." }), cancel = screen.getByRole("button", { name: "Cancel" }); expect(save).toBeDisabled(); expect(cancel).toBeDisabled(); fireEvent.click(save); fireEvent.click(cancel); expect(submit).not.toHaveBeenCalled(); expect(close).not.toHaveBeenCalled(); expect(screen.getByLabelText("Title")).toHaveValue("Pending draft");
  });
  it("cancels without submitting then reopens with clean selected-kind defaults", () => {
    const submit = vi.fn(), close = vi.fn(); const view = renderWithProviders(<BacklogFormDialog isOpen mode="create" defaultKind="chore" onSubmit={submit} onClose={close} />); change("Title", "Abandoned chore"); change("Name", "abandoned-identity"); fireEvent.click(screen.getByRole("button", { name: "Cancel" })); expect(close).toHaveBeenCalledOnce(); expect(submit).not.toHaveBeenCalled();
    view.rerender(<BacklogFormDialog isOpen={false} mode="create" defaultKind="fix" onSubmit={submit} onClose={close} />); view.rerender(<BacklogFormDialog isOpen mode="create" defaultKind="fix" onSubmit={submit} onClose={close} />);
    expect(screen.getByLabelText("Title")).toHaveValue(""); expect(screen.getByLabelText("Name")).toHaveValue(""); expect(screen.getByLabelText("Backlog type")).toHaveValue("fix"); expect(useBacklogFormStore.getState().nameDirty).toBe(false); expect(useBacklogFormStore.getState().error).toBeNull();
  });
});

// Desired real-caller compatibility: equal owner values are recreated inline.
describe("Backlog equivalent-owner-prop draft preservation", () => {
  it("keeps an edited title across a fresh equal-semantic initialValues object", () => {
    const submit = vi.fn(), close = vi.fn();
    const view = renderWithProviders(<BacklogFormDialog isOpen mode="edit" initialValues={original} onSubmit={submit} onClose={close} />);
    change("Title", "Unsaved operator backlog draft");
    view.rerender(<BacklogFormDialog isOpen mode="edit" initialValues={{ ...original }} onSubmit={submit} onClose={close} />);
    expect(screen.getByLabelText("Title")).toHaveValue("Unsaved operator backlog draft");
    expect(screen.getByLabelText("Name")).toHaveValue("owner-work"); expect(submit).not.toHaveBeenCalled(); expect(close).not.toHaveBeenCalled();
  });
});

describe("Backlog complete semantic initialization lifecycle", () => {
  it("retains refused and pending drafts across equivalent nested values and limit key order", () => {
    const submit = vi.fn(), close = vi.fn(); const limits = { maxSlices: 4, maxTokens: 500, maxWallSeconds: 60, maxTurns: 9, maxChargeMicroUsd: 2000, maxChildren: 2, maxNodeAttempts: 3, maxRetries: 1 };
    const initial = { ...original, executionLimits: limits };
    const view = renderWithProviders(<BacklogFormDialog isOpen mode="edit" initialValues={initial} onSubmit={submit} onClose={close} />); change("Title", "Retained nested draft");
    const reordered = { maxRetries: 1, maxNodeAttempts: 3, maxChildren: 2, maxChargeMicroUsd: 2000, maxTurns: 9, maxWallSeconds: 60, maxTokens: 500, maxSlices: 4 };
    view.rerender(<BacklogFormDialog isOpen mode="edit" initialValues={{ ...initial, tags: [...initial.tags], dependsOn: [...initial.dependsOn!], acceptanceAllow: [...initial.acceptanceAllow!], acceptanceDeny: [...initial.acceptanceDeny!], executionLimits: reordered }} isSubmitting submitError="Owner refusal" onSubmit={submit} onClose={close} />);
    expect(screen.getByLabelText("Title")).toHaveValue("Retained nested draft"); expect(screen.getByText("Owner refusal")).toBeVisible(); expect(screen.getByRole("button", { name: "Saving..." })).toBeDisabled(); expect(submit).not.toHaveBeenCalled();
  });
  it("treats omitted optional values and explicit store defaults as equivalent", () => {
    const submit = vi.fn(), close = vi.fn(); const initial = { name: "owner-work", title: "Owner work", description: "", status: "backlog" as const, priority: 5, tags: [], kind: "idea" as const };
    const view = renderWithProviders(<BacklogFormDialog isOpen mode="edit" initialValues={initial} onSubmit={submit} onClose={close} />); change("Title", "Retained default draft");
    view.rerender(<BacklogFormDialog isOpen mode="edit" initialValues={{ ...initial, dependsOn: [], milestone: "", effort: "", acceptanceAllow: [], acceptanceDeny: [], executionMode: "sliced", continuation: "manual", scopePolicy: "fixed" }} onSubmit={submit} onClose={close} />);
    expect(screen.getByLabelText("Title")).toHaveValue("Retained default draft"); expect(submit).not.toHaveBeenCalled();
  });
  it("loads changed canonical arrays, execution limits, continuation and scope values", () => {
    const submit = vi.fn(), close = vi.fn(); const view = renderWithProviders(<BacklogFormDialog isOpen mode="edit" initialValues={original} onSubmit={submit} onClose={close} />); change("Title", "Local draft");
    const changed = { ...original, tags: ["canonical"], dependsOn: ["fix/new-owner"], acceptanceAllow: ["docs/**"], acceptanceDeny: ["secrets/**"], executionLimits: { maxSlices: 2, maxTokens: 300, maxWallSeconds: 45, maxTurns: 4, maxChargeMicroUsd: 1000, maxChildren: 1, maxNodeAttempts: 2, maxRetries: 0 }, continuation: "until-allowance", scopePolicy: "extend-with-record" };
    view.rerender(<BacklogFormDialog isOpen mode="edit" initialValues={changed} onSubmit={submit} onClose={close} />); expect(screen.getByLabelText("Title")).toHaveValue("Owner work"); expect(screen.getByLabelText("Dependencies")).toHaveValue("fix/new-owner"); expect(screen.getByLabelText("Maximum slices")).toHaveValue(2); expect(useBacklogFormStore.getState().values).toEqual({ ...changed, milestone: "owner-milestone", effort: "S", executionMode: "sliced" }); expect(submit).not.toHaveBeenCalled();
  });
  it("loads a new owner and reinitializes on mode and selected default-kind changes", () => {
    const submit = vi.fn(), close = vi.fn(); const view = renderWithProviders(<BacklogFormDialog isOpen mode="edit" initialValues={original} onSubmit={submit} onClose={close} />); change("Title", "Local draft");
    view.rerender(<BacklogFormDialog isOpen mode="edit" initialValues={{ ...original, name: "other-work", title: "Other work" }} onSubmit={submit} onClose={close} />); expect(screen.getByLabelText("Name")).toHaveValue("other-work"); expect(screen.getByLabelText("Title")).toHaveValue("Other work");
    view.rerender(<BacklogFormDialog isOpen mode="create" defaultKind="research" onSubmit={submit} onClose={close} />); expect(screen.getByLabelText("Title")).toHaveValue(""); expect(screen.getByLabelText("Backlog type")).toHaveValue("research"); change("Title", "New research draft");
    view.rerender(<BacklogFormDialog isOpen mode="create" defaultKind="chore" onSubmit={submit} onClose={close} />); expect(screen.getByLabelText("Title")).toHaveValue(""); expect(screen.getByLabelText("Backlog type")).toHaveValue("chore"); expect(submit).not.toHaveBeenCalled();
  });
});
