import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen } from "@testing-library/react";
import { getSpatialNav } from "@vrooli/iframe-bridge/spatial";
import { renderWithProviders } from "../../test-utils";
import { defaultApiClient } from "../../lib/api-client";
import { RequirementFormDialog } from "./requirement-form-dialog";
import { TargetFormDialog } from "./target-form-dialog";
import { ModuleFormDialog } from "./module-form-dialog";

// Actual forms and children. Callbacks are parent-owned mutation boundaries.
beforeEach(() => {
  for (const method of ["get", "post", "put", "patch", "delete"] as const) vi.spyOn(defaultApiClient, method).mockRejectedValue(new Error(`Unexpected archive form network ${method}`));
  vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("Unexpected archive form fetch")));
});
afterEach(() => { cleanup(); getSpatialNav()?.dispose(); for (const method of ["get", "post", "put", "patch", "delete"] as const) expect(defaultApiClient[method]).not.toHaveBeenCalled(); expect(fetch).not.toHaveBeenCalled(); vi.restoreAllMocks(); vi.unstubAllGlobals(); });
function change(label: string, value: string) { fireEvent.change(screen.getByLabelText(label), { target: { value } }); }

describe("Requirement form callback workflows", () => {
  it("validates required values before offering a parent mutation", () => {
    const submit = vi.fn(), close = vi.fn(); renderWithProviders(<RequirementFormDialog isOpen mode="create" onSubmit={submit} onClose={close} />);
    fireEvent.click(screen.getByRole("button", { name: "Create Requirement" })); expect(screen.getByText("ID is required.")).toBeVisible(); expect(submit).not.toHaveBeenCalled();
    change("ID", "REQ-NEW"); change("Title", "   "); fireEvent.click(screen.getByRole("button", { name: "Create Requirement" })); expect(screen.getByText("Title is required.")).toBeVisible(); expect(submit).not.toHaveBeenCalled(); expect(close).not.toHaveBeenCalled();
  });
  it("submits trimmed explicit fields and parsed current references exactly once", () => {
    const submit = vi.fn(), close = vi.fn(); renderWithProviders(<RequirementFormDialog isOpen mode="create" onSubmit={submit} onClose={close} />);
    change("ID", " REQ-NEW "); change("Title", " New requirement "); change("Description", " Explicit description "); change("Status", "complete"); change("Category", " functional "); change("PRD Reference", " PRD-NEW "); change("Notes", " Reviewed notes "); fireEvent.click(screen.getByRole("button", { name: "Create Requirement" })); expect(submit.mock.calls).toEqual([[{ id: "REQ-NEW", title: "New requirement", description: "Explicit description", status: "complete", category: "functional", prd_ref: "PRD-NEW", notes: "Reviewed notes" }]]); expect(close).not.toHaveBeenCalled();
  });
  it("preserves optional defaults rather than inventing identity or references", () => {
    const submit = vi.fn(); renderWithProviders(<RequirementFormDialog isOpen mode="create" onSubmit={submit} onClose={vi.fn()} />); change("ID", "REQ-NEW"); change("Title", " Minimal title "); fireEvent.click(screen.getByRole("button", { name: "Create Requirement" })); expect(submit.mock.calls).toEqual([[{ id: "REQ-NEW", title: "Minimal title", description: "", status: "pending", category: "", prd_ref: "", notes: undefined }]]);
  });
  it("locks existing identity while allowing exact edited owner values", () => {
    const submit = vi.fn(); const original = { id: "REQ-OWNER", title: "Existing requirement", description: "Owner description", status: "in_progress", category: "functional", prd_ref: "PRD-OWNER", notes: "Owner notes" }; renderWithProviders(<RequirementFormDialog isOpen mode="edit" initialValues={original} onSubmit={submit} onClose={vi.fn()} />);
    expect(screen.getByLabelText("ID")).toBeDisabled(); expect(screen.getByLabelText("ID")).toHaveValue(original.id); change("Title", " Edited owner title "); fireEvent.click(screen.getByRole("button", { name: "Save Changes" })); expect(submit.mock.calls).toEqual([[{ ...original, title: "Edited owner title" }]]);
  });
  it("retains the draft and displays a parent refusal without silently resubmitting", () => {
    const submit = vi.fn(), close = vi.fn(); const original = { id: "REQ-OWNER", title: "Existing requirement", description: "Owner description", status: "in_progress", category: "functional", prd_ref: "PRD-OWNER", notes: "Owner notes" }; const view = renderWithProviders(<RequirementFormDialog isOpen mode="edit" initialValues={original} onSubmit={submit} onClose={close} />);
    change("Title", "Unaccepted draft"); view.rerender(<RequirementFormDialog isOpen mode="edit" initialValues={original} submitError="Owner refused mutation" onSubmit={submit} onClose={close} />); expect(screen.getByText("Owner refused mutation")).toBeVisible(); expect(screen.getByLabelText("Title")).toHaveValue("Unaccepted draft"); expect(submit).not.toHaveBeenCalled();
  });
  it("disables real inputs, submit and explicit Cancel while the parent is submitting", () => {
    const submit = vi.fn(), close = vi.fn(); const original = { id: "REQ-OWNER", title: "Existing requirement", description: "Owner description", status: "in_progress", category: "functional", prd_ref: "PRD-OWNER", notes: "Owner notes" }; const view = renderWithProviders(<RequirementFormDialog isOpen mode="edit" initialValues={original} onSubmit={submit} onClose={close} />); change("Title", "Pending draft");
    view.rerender(<RequirementFormDialog isOpen mode="edit" initialValues={original} isSubmitting onSubmit={submit} onClose={close} />);
    for (const field of [...screen.getAllByRole("textbox"), ...screen.queryAllByRole("combobox")]) expect(field).toBeDisabled(); const save = screen.getByRole("button", { name: "Saving..." }), cancel = screen.getByRole("button", { name: "Cancel" }); expect(save).toBeDisabled(); expect(cancel).toBeDisabled(); fireEvent.click(save); fireEvent.click(cancel); expect(submit).not.toHaveBeenCalled(); expect(close).not.toHaveBeenCalled(); expect(screen.getByLabelText("Title")).toHaveValue("Pending draft");
  });
  it("cancels without submission and resets prior draft and validation when reopened", () => {
    const submit = vi.fn(), close = vi.fn(); const view = renderWithProviders(<RequirementFormDialog isOpen mode="create" onSubmit={submit} onClose={close} />); change("Title", "Abandoned draft"); fireEvent.click(screen.getByRole("button", { name: "Cancel" })); expect(close).toHaveBeenCalledOnce(); expect(submit).not.toHaveBeenCalled();
    view.rerender(<RequirementFormDialog isOpen={false} mode="create" onSubmit={submit} onClose={close} />); view.rerender(<RequirementFormDialog isOpen mode="create" onSubmit={submit} onClose={close} />); expect(screen.getByLabelText("Title")).toHaveValue(""); expect(screen.getByLabelText("ID")).toHaveValue("");
  });
});

describe("Target form callback workflows", () => {
  it("validates required values before offering a parent mutation", () => {
    const submit = vi.fn(), close = vi.fn(); renderWithProviders(<TargetFormDialog isOpen mode="create" onSubmit={submit} onClose={close} />);
    fireEvent.click(screen.getByRole("button", { name: "Create Target" })); expect(screen.getByText("Title is required.")).toBeVisible(); expect(submit).not.toHaveBeenCalled();
     change("Title", "   "); fireEvent.click(screen.getByRole("button", { name: "Create Target" })); expect(screen.getByText("Title is required.")).toBeVisible(); expect(submit).not.toHaveBeenCalled(); expect(close).not.toHaveBeenCalled();
  });
  it("submits trimmed explicit fields and parsed current references exactly once", () => {
    const submit = vi.fn(), close = vi.fn(); renderWithProviders(<TargetFormDialog isOpen mode="create" onSubmit={submit} onClose={close} />);
    change("ID", " OT-NEW "); change("Title", " New target "); change("Criticality", "P2"); change("Status", "complete"); change("Notes", " Reviewed notes "); change("Linked Requirements", " REQ-A, ,REQ-B , "); fireEvent.click(screen.getByRole("button", { name: "Create Target" })); expect(submit.mock.calls).toEqual([[{ id: "OT-NEW", title: "New target", criticality: "P2", status: "complete", notes: "Reviewed notes", linked_requirement_ids: ["REQ-A", "REQ-B"] }]]); expect(close).not.toHaveBeenCalled();
  });
  it("preserves optional defaults rather than inventing identity or references", () => {
    const submit = vi.fn(); renderWithProviders(<TargetFormDialog isOpen mode="create" onSubmit={submit} onClose={vi.fn()} />);  change("Title", " Minimal title "); fireEvent.click(screen.getByRole("button", { name: "Create Target" })); expect(submit.mock.calls).toEqual([[{ id: "", title: "Minimal title", criticality: "P0", status: "pending", notes: "", linked_requirement_ids: [] }]]);
  });
  it("locks existing identity while allowing exact edited owner values", () => {
    const submit = vi.fn(); const original = { id: "OT-OWNER", title: "Existing target", criticality: "P1", status: "complete", notes: "Owner notes", linked_requirement_ids: ["REQ-A", "REQ-B"] }; renderWithProviders(<TargetFormDialog isOpen mode="edit" initialValues={original} onSubmit={submit} onClose={vi.fn()} />);
    expect(screen.getByLabelText("ID")).toBeDisabled(); expect(screen.getByLabelText("ID")).toHaveValue(original.id); change("Title", " Edited owner title "); fireEvent.click(screen.getByRole("button", { name: "Save Changes" })); expect(submit.mock.calls).toEqual([[{ ...original, title: "Edited owner title" }]]);
  });
  it("retains the draft and displays a parent refusal without silently resubmitting", () => {
    const submit = vi.fn(), close = vi.fn(); const original = { id: "OT-OWNER", title: "Existing target", criticality: "P1", status: "complete", notes: "Owner notes", linked_requirement_ids: ["REQ-A", "REQ-B"] }; const view = renderWithProviders(<TargetFormDialog isOpen mode="edit" initialValues={original} onSubmit={submit} onClose={close} />);
    change("Title", "Unaccepted draft"); view.rerender(<TargetFormDialog isOpen mode="edit" initialValues={original} submitError="Owner refused mutation" onSubmit={submit} onClose={close} />); expect(screen.getByText("Owner refused mutation")).toBeVisible(); expect(screen.getByLabelText("Title")).toHaveValue("Unaccepted draft"); expect(submit).not.toHaveBeenCalled();
  });
  it("disables real inputs, submit and explicit Cancel while the parent is submitting", () => {
    const submit = vi.fn(), close = vi.fn(); const original = { id: "OT-OWNER", title: "Existing target", criticality: "P1", status: "complete", notes: "Owner notes", linked_requirement_ids: ["REQ-A", "REQ-B"] }; const view = renderWithProviders(<TargetFormDialog isOpen mode="edit" initialValues={original} onSubmit={submit} onClose={close} />); change("Title", "Pending draft");
    view.rerender(<TargetFormDialog isOpen mode="edit" initialValues={original} isSubmitting onSubmit={submit} onClose={close} />);
    for (const field of [...screen.getAllByRole("textbox"), ...screen.queryAllByRole("combobox")]) expect(field).toBeDisabled(); const save = screen.getByRole("button", { name: "Saving..." }), cancel = screen.getByRole("button", { name: "Cancel" }); expect(save).toBeDisabled(); expect(cancel).toBeDisabled(); fireEvent.click(save); fireEvent.click(cancel); expect(submit).not.toHaveBeenCalled(); expect(close).not.toHaveBeenCalled(); expect(screen.getByLabelText("Title")).toHaveValue("Pending draft");
  });
  it("cancels without submission and resets prior draft and validation when reopened", () => {
    const submit = vi.fn(), close = vi.fn(); const view = renderWithProviders(<TargetFormDialog isOpen mode="create" onSubmit={submit} onClose={close} />); change("Title", "Abandoned draft"); fireEvent.click(screen.getByRole("button", { name: "Cancel" })); expect(close).toHaveBeenCalledOnce(); expect(submit).not.toHaveBeenCalled();
    view.rerender(<TargetFormDialog isOpen={false} mode="create" onSubmit={submit} onClose={close} />); view.rerender(<TargetFormDialog isOpen mode="create" onSubmit={submit} onClose={close} />); expect(screen.getByLabelText("Title")).toHaveValue(""); expect(screen.getByLabelText("ID")).toHaveValue("");
  });
});

describe("Module form callback workflows", () => {
  it("validates required values before offering a parent mutation", () => {
    const submit = vi.fn(), close = vi.fn(); renderWithProviders(<ModuleFormDialog isOpen mode="create" onSubmit={submit} onClose={close} />);
    fireEvent.click(screen.getByRole("button", { name: "Create Module" })); expect(screen.getByText("ID is required.")).toBeVisible(); expect(submit).not.toHaveBeenCalled();
    change("ID", "new-module"); change("Title", "   "); fireEvent.click(screen.getByRole("button", { name: "Create Module" })); expect(screen.getByText("Title is required.")).toBeVisible(); expect(submit).not.toHaveBeenCalled(); expect(close).not.toHaveBeenCalled();
  });
  it("submits trimmed explicit fields and parsed current references exactly once", () => {
    const submit = vi.fn(), close = vi.fn(); renderWithProviders(<ModuleFormDialog isOpen mode="create" onSubmit={submit} onClose={close} />);
    change("ID", " new-module "); change("Title", " New module "); change("Description", " Explicit description "); fireEvent.click(screen.getByRole("button", { name: "Create Module" })); expect(submit.mock.calls).toEqual([[{ id: "new-module", title: "New module", description: "Explicit description" }]]); expect(close).not.toHaveBeenCalled();
  });
  it("preserves optional defaults rather than inventing identity or references", () => {
    const submit = vi.fn(); renderWithProviders(<ModuleFormDialog isOpen mode="create" onSubmit={submit} onClose={vi.fn()} />); change("ID", "new-module"); change("Title", " Minimal title "); fireEvent.click(screen.getByRole("button", { name: "Create Module" })); expect(submit.mock.calls).toEqual([[{ id: "new-module", title: "Minimal title", description: "" }]]);
  });
  it("locks existing identity while allowing exact edited owner values", () => {
    const submit = vi.fn(); const original = { id: "owner-module", title: "Existing module", description: "Owner description" }; renderWithProviders(<ModuleFormDialog isOpen mode="edit" initialValues={original} onSubmit={submit} onClose={vi.fn()} />);
    expect(screen.getByLabelText("ID")).toBeDisabled(); expect(screen.getByLabelText("ID")).toHaveValue(original.id); change("Title", " Edited owner title "); fireEvent.click(screen.getByRole("button", { name: "Save Changes" })); expect(submit.mock.calls).toEqual([[{ ...original, title: "Edited owner title" }]]);
  });
  it("retains the draft and displays a parent refusal without silently resubmitting", () => {
    const submit = vi.fn(), close = vi.fn(); const original = { id: "owner-module", title: "Existing module", description: "Owner description" }; const view = renderWithProviders(<ModuleFormDialog isOpen mode="edit" initialValues={original} onSubmit={submit} onClose={close} />);
    change("Title", "Unaccepted draft"); view.rerender(<ModuleFormDialog isOpen mode="edit" initialValues={original} submitError="Owner refused mutation" onSubmit={submit} onClose={close} />); expect(screen.getByText("Owner refused mutation")).toBeVisible(); expect(screen.getByLabelText("Title")).toHaveValue("Unaccepted draft"); expect(submit).not.toHaveBeenCalled();
  });
  it("disables real inputs, submit and explicit Cancel while the parent is submitting", () => {
    const submit = vi.fn(), close = vi.fn(); const original = { id: "owner-module", title: "Existing module", description: "Owner description" }; const view = renderWithProviders(<ModuleFormDialog isOpen mode="edit" initialValues={original} onSubmit={submit} onClose={close} />); change("Title", "Pending draft");
    view.rerender(<ModuleFormDialog isOpen mode="edit" initialValues={original} isSubmitting onSubmit={submit} onClose={close} />);
    for (const field of [...screen.getAllByRole("textbox"), ...screen.queryAllByRole("combobox")]) expect(field).toBeDisabled(); const save = screen.getByRole("button", { name: "Saving..." }), cancel = screen.getByRole("button", { name: "Cancel" }); expect(save).toBeDisabled(); expect(cancel).toBeDisabled(); fireEvent.click(save); fireEvent.click(cancel); expect(submit).not.toHaveBeenCalled(); expect(close).not.toHaveBeenCalled(); expect(screen.getByLabelText("Title")).toHaveValue("Pending draft");
  });
  it("cancels without submission and resets prior draft and validation when reopened", () => {
    const submit = vi.fn(), close = vi.fn(); const view = renderWithProviders(<ModuleFormDialog isOpen mode="create" onSubmit={submit} onClose={close} />); change("Title", "Abandoned draft"); fireEvent.click(screen.getByRole("button", { name: "Cancel" })); expect(close).toHaveBeenCalledOnce(); expect(submit).not.toHaveBeenCalled();
    view.rerender(<ModuleFormDialog isOpen={false} mode="create" onSubmit={submit} onClose={close} />); view.rerender(<ModuleFormDialog isOpen mode="create" onSubmit={submit} onClose={close} />); expect(screen.getByLabelText("Title")).toHaveValue(""); expect(screen.getByLabelText("ID")).toHaveValue("");
  });
});

// Desired real-caller compatibility: BacklogDialogs reconstructs these values
// on parent renders. Equivalent owner data must not erase an unsaved draft.
describe("Module equivalent-owner-prop draft preservation", () => {
  it("keeps an edited title across a fresh equal-semantic initialValues object", () => {
    const submit = vi.fn(), close = vi.fn();
    const initial = { id: "owner-module", title: "Owner module", description: "Owner description" };
    const view = renderWithProviders(<ModuleFormDialog isOpen mode="edit" initialValues={initial} onSubmit={submit} onClose={close} />);
    change("Title", "Unsaved operator module draft");
    view.rerender(<ModuleFormDialog isOpen mode="edit" initialValues={{ ...initial }} onSubmit={submit} onClose={close} />);
    expect(screen.getByLabelText("Title")).toHaveValue("Unsaved operator module draft");
    expect(screen.getByLabelText("ID")).toHaveValue("owner-module"); expect(submit).not.toHaveBeenCalled(); expect(close).not.toHaveBeenCalled();
  });
});

describe("Module semantic initialization lifecycle", () => {
  it("retains refused and pending drafts on equivalent parent values", () => {
    const submit = vi.fn(), close = vi.fn(); const initial = { id: "owner-module", title: "Owner module", description: "Owner description" };
    const view = renderWithProviders(<ModuleFormDialog isOpen mode="edit" initialValues={initial} onSubmit={submit} onClose={close} />);
    change("Title", "Retained module draft"); view.rerender(<ModuleFormDialog isOpen mode="edit" initialValues={{ ...initial }} submitError="Owner refusal" isSubmitting onSubmit={submit} onClose={close} />);
    expect(screen.getByLabelText("Title")).toHaveValue("Retained module draft"); expect(screen.getByText("Owner refusal")).toBeVisible(); expect(screen.getByRole("button", { name: "Saving..." })).toBeDisabled(); expect(submit).not.toHaveBeenCalled();
  });
  it("loads changed canonical owner values and a newly selected module", () => {
    const submit = vi.fn(), close = vi.fn(); const initial = { id: "owner-module", title: "Owner module", description: "Owner description" };
    const view = renderWithProviders(<ModuleFormDialog isOpen mode="edit" initialValues={initial} onSubmit={submit} onClose={close} />); change("Title", "Local draft");
    view.rerender(<ModuleFormDialog isOpen mode="edit" initialValues={{ ...initial, title: "Canonical module" }} onSubmit={submit} onClose={close} />); expect(screen.getByLabelText("Title")).toHaveValue("Canonical module");
    view.rerender(<ModuleFormDialog isOpen mode="edit" initialValues={{ id: "other-module", title: "Other module", description: "Other description" }} onSubmit={submit} onClose={close} />); expect(screen.getByLabelText("ID")).toHaveValue("other-module"); expect(screen.getByLabelText("Description")).toHaveValue("Other description"); expect(submit).not.toHaveBeenCalled();
  });
  it("reinitializes when mode changes even when owner primitives are unchanged", () => {
    const submit = vi.fn(), close = vi.fn(); const initial = { id: "owner-module", title: "Owner module", description: "Owner description" };
    const view = renderWithProviders(<ModuleFormDialog isOpen mode="edit" initialValues={initial} onSubmit={submit} onClose={close} />); change("Title", "Local draft");
    view.rerender(<ModuleFormDialog isOpen mode="create" initialValues={initial} onSubmit={submit} onClose={close} />); expect(screen.getByLabelText("Title")).toHaveValue("Owner module"); expect(screen.getByLabelText("ID")).not.toBeDisabled(); expect(submit).not.toHaveBeenCalled();
  });
});
