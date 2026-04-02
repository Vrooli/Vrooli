import { describe, it, expect, beforeEach } from "vitest";
import { useBacklogDetailUIStore } from "./backlog-detail-ui-store";

describe("backlog-detail-ui-store", () => {
  beforeEach(() => {
    useBacklogDetailUIStore.getState().reset();
  });

  // --- Simple dialog toggles ---

  it("opens and closes edit dialog", () => {
    const store = useBacklogDetailUIStore;
    store.getState().openEdit();
    expect(store.getState().showEdit).toBe(true);
    store.getState().closeEdit();
    expect(store.getState().showEdit).toBe(false);
  });

  it("opens and closes delete dialog", () => {
    const store = useBacklogDetailUIStore;
    store.getState().openDelete();
    expect(store.getState().showDelete).toBe(true);
    store.getState().closeDelete();
    expect(store.getState().showDelete).toBe(false);
  });

  it("opens and closes agent dialog", () => {
    const store = useBacklogDetailUIStore;
    store.getState().openAgent();
    expect(store.getState().showAgentDialog).toBe(true);
    store.getState().closeAgent();
    expect(store.getState().showAgentDialog).toBe(false);
  });

  it("opens and closes run modal", () => {
    const store = useBacklogDetailUIStore;
    store.getState().openRunModal();
    expect(store.getState().showRunModal).toBe(true);
    store.getState().closeRunModal();
    expect(store.getState().showRunModal).toBe(false);
  });

  it("opens and closes timeline", () => {
    const store = useBacklogDetailUIStore;
    store.getState().openTimeline();
    expect(store.getState().isTimelineOpen).toBe(true);
    store.getState().closeTimeline();
    expect(store.getState().isTimelineOpen).toBe(false);
  });

  // --- Follow-up target ---

  it("sets and clears follow-up target", () => {
    const store = useBacklogDetailUIStore;
    const exec = { id: "exec-1" } as any;
    store.getState().setFollowUpTarget(exec);
    expect(store.getState().followUpTarget).toBe(exec);
    store.getState().setFollowUpTarget(null);
    expect(store.getState().followUpTarget).toBeNull();
  });

  // --- Round to delete ---

  it("sets and clears round to delete", () => {
    const store = useBacklogDetailUIStore;
    store.getState().setRoundToDelete(3);
    expect(store.getState().roundToDelete).toBe(3);
    store.getState().setRoundToDelete(null);
    expect(store.getState().roundToDelete).toBeNull();
  });

  // --- CRUD dialogs ---

  it("opens target dialog in create mode", () => {
    const store = useBacklogDetailUIStore;
    store.getState().openTargetCreate();
    expect(store.getState().targetDialog).toEqual({ isOpen: true, mode: "create", editing: null });
  });

  it("opens target dialog in edit mode", () => {
    const store = useBacklogDetailUIStore;
    const target = { id: "t1", title: "Target 1" } as any;
    store.getState().openTargetEdit(target);
    expect(store.getState().targetDialog).toEqual({ isOpen: true, mode: "edit", editing: target });
  });

  it("closes target dialog", () => {
    const store = useBacklogDetailUIStore;
    store.getState().openTargetCreate();
    store.getState().closeTargetDialog();
    expect(store.getState().targetDialog.isOpen).toBe(false);
  });

  it("opens req dialog in create mode", () => {
    const store = useBacklogDetailUIStore;
    store.getState().openReqCreate("mod1");
    expect(store.getState().reqDialog).toEqual({ isOpen: true, mode: "create", editing: { groupId: "mod1" } });
  });

  it("opens req dialog in edit mode", () => {
    const store = useBacklogDetailUIStore;
    const data = { groupId: "mod1", req: { id: "r1" } as any };
    store.getState().openReqEdit(data);
    expect(store.getState().reqDialog).toEqual({ isOpen: true, mode: "edit", editing: data });
  });

  it("opens module dialog in create/edit modes", () => {
    const store = useBacklogDetailUIStore;
    store.getState().openModuleCreate();
    expect(store.getState().moduleDialog).toEqual({ isOpen: true, mode: "create", editing: null });
    store.getState().openModuleEdit("mod1");
    expect(store.getState().moduleDialog).toEqual({ isOpen: true, mode: "edit", editing: "mod1" });
  });

  // --- Selection ---

  it("toggles target id selection", () => {
    const store = useBacklogDetailUIStore;
    store.getState().toggleTargetId("t1");
    expect(store.getState().selectedTargetIds.has("t1")).toBe(true);
    store.getState().toggleTargetId("t1");
    expect(store.getState().selectedTargetIds.has("t1")).toBe(false);
  });

  it("toggles requirement id selection", () => {
    const store = useBacklogDetailUIStore;
    store.getState().toggleRequirementId("r1");
    expect(store.getState().selectedRequirementIds.has("r1")).toBe(true);
    store.getState().toggleRequirementId("r1");
    expect(store.getState().selectedRequirementIds.has("r1")).toBe(false);
  });

  it("clears all selections", () => {
    const store = useBacklogDetailUIStore;
    store.getState().toggleTargetId("t1");
    store.getState().toggleRequirementId("r1");
    store.getState().clearSelections();
    expect(store.getState().selectedTargetIds.size).toBe(0);
    expect(store.getState().selectedRequirementIds.size).toBe(0);
  });

  // --- Review mode ---

  it("toggles review mode", () => {
    const store = useBacklogDetailUIStore;
    expect(store.getState().reviewMode).toBe(false);
    store.getState().toggleReviewMode();
    expect(store.getState().reviewMode).toBe(true);
    store.getState().toggleReviewMode();
    expect(store.getState().reviewMode).toBe(false);
  });

  // --- Reset ---

  it("resets all state", () => {
    const store = useBacklogDetailUIStore;
    store.getState().openEdit();
    store.getState().openAgent();
    store.getState().toggleTargetId("t1");
    store.getState().toggleReviewMode();
    store.getState().setRoundToDelete(2);
    store.getState().openTargetCreate();

    store.getState().reset();

    const state = store.getState();
    expect(state.showEdit).toBe(false);
    expect(state.showAgentDialog).toBe(false);
    expect(state.selectedTargetIds.size).toBe(0);
    expect(state.reviewMode).toBe(false);
    expect(state.roundToDelete).toBeNull();
    expect(state.targetDialog.isOpen).toBe(false);
  });
});
