import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useBacklogHandlers, type UseBacklogHandlersOptions } from "./useBacklogHandlers";
import { useBacklogDetailUIStore } from "../stores";

// Shape matching the mock data so we can access _mutations without `any`
interface MockMutation { mutate: ReturnType<typeof vi.fn>; isPending: boolean; isError: boolean; error: null; reset: ReturnType<typeof vi.fn> }
interface MockMutations { update: MockMutation; acceptanceGlob: MockMutation; delete: MockMutation; agent: MockMutation; workshopSave: MockMutation; workshopDeleteRound: MockMutation; fileAction: MockMutation; updateReqs: MockMutation; createModule: MockMutation; updateModuleMeta: MockMutation; createTarget: MockMutation; updateTarget: MockMutation }
interface MockData { _mutations: MockMutations; updateRequirements: ReturnType<typeof vi.fn>; [key: string]: unknown }

// Minimal mock for _mutations
const makeMockMutation = (): MockMutation => ({
  mutate: vi.fn(),
  isPending: false,
  isError: false,
  error: null,
  reset: vi.fn(),
});

const makeMockData = () => ({
  _mutations: {
    update: makeMockMutation(),
    acceptanceGlob: makeMockMutation(),
    delete: makeMockMutation(),
    agent: makeMockMutation(),
    workshopSave: makeMockMutation(),
    workshopDeleteRound: makeMockMutation(),
    fileAction: makeMockMutation(),
    updateReqs: makeMockMutation(),
    createModule: makeMockMutation(),
    updateModuleMeta: makeMockMutation(),
    createTarget: makeMockMutation(),
    updateTarget: makeMockMutation(),
  },
  archiveTargets: {
    targets: [],
    requirements: [
      { id: "mod1", name: "Module 1", requirements: [{ id: "req1", title: "Req 1" }], children: [] },
    ],
    has_archive: true,
  },
  targetIdSet: new Set(["t1"]),
  reqModuleMap: new Map([["req1", "mod1"]]),
  deliverableLabelLower: "plan",
  invalidateFiles: vi.fn(),
  deleteTarget: vi.fn(),
  deleteModule: vi.fn(),
  updateRequirements: vi.fn(),
  batchReview: vi.fn(),
});

function makeOpts(overrides: Partial<UseBacklogHandlersOptions> = {}): UseBacklogHandlersOptions {
  return {
    data: makeMockData() as unknown as UseBacklogHandlersOptions["data"],
    backlogKind: "idea",
    name: "test-item",
    setSelectedFile: vi.fn(),
    setActiveTab: vi.fn(),
    setSearchParams: vi.fn(),
    selectedFile: null,
    closeDetail: vi.fn(),
    refreshActivities: vi.fn().mockResolvedValue(undefined),
    ...overrides,
  };
}

describe("useBacklogHandlers", () => {
  beforeEach(() => {
    useBacklogDetailUIStore.getState().reset();
  });

  it("handleUpdateItem calls update mutation", () => {
    const opts = makeOpts();
    const { result } = renderHook(() => useBacklogHandlers(opts));

    act(() => {
      result.current.handleUpdateItem({
        title: "T", description: "D", status: "ready", priority: 1, tags: [],
      });
    });

    expect((opts.data as unknown as MockData)._mutations.update.mutate).toHaveBeenCalledWith(
      { title: "T", description: "D", status: "ready", priority: 1, tags: [] },
       
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );
  });

  it("handleDeleteConfirm calls delete mutation", () => {
    const opts = makeOpts();
    const { result } = renderHook(() => useBacklogHandlers(opts));

    act(() => {
      result.current.handleDeleteConfirm();
    });

    expect((opts.data as unknown as MockData)._mutations.delete.mutate).toHaveBeenCalled();
  });

  it("handleRunWorkshop calls agent mutation with workshop mode", () => {
    const opts = makeOpts();
    const { result } = renderHook(() => useBacklogHandlers(opts));

    act(() => {
      result.current.handleRunWorkshop();
    });

    expect((opts.data as unknown as MockData)._mutations.agent.mutate).toHaveBeenCalledWith(
      expect.objectContaining({ mode: "workshop" }),
      expect.any(Object),
    );
  });

  it("handleFinalizeWorkshop calls agent mutation with finalize mode", () => {
    const opts = makeOpts();
    const { result } = renderHook(() => useBacklogHandlers(opts));

    act(() => {
      result.current.handleFinalizeWorkshop();
    });

    expect((opts.data as unknown as MockData)._mutations.agent.mutate).toHaveBeenCalledWith(
      expect.objectContaining({ mode: "finalize" }),
      expect.any(Object),
    );
  });

  it("handleCreateTarget opens target dialog in store", () => {
    const opts = makeOpts();
    const { result } = renderHook(() => useBacklogHandlers(opts));

    act(() => {
      result.current.handleCreateTarget();
    });

    const state = useBacklogDetailUIStore.getState();
    expect(state.targetDialog.isOpen).toBe(true);
    expect(state.targetDialog.mode).toBe("create");
  });

  it("handleDeleteRequirement uses findRequirementGroup to locate the group", () => {
    vi.spyOn(window, "confirm").mockReturnValue(true);

    const opts = makeOpts();
    const { result } = renderHook(() => useBacklogHandlers(opts));

    act(() => {
      result.current.handleDeleteRequirement("mod1", "req1");
    });

    expect((opts.data as unknown as MockData).updateRequirements).toHaveBeenCalledWith({
      moduleId: "mod1",
      requirements: [],
    });

    vi.restoreAllMocks();
  });

  it("handleTargetToggle toggles id in store", () => {
    const opts = makeOpts();
    const { result } = renderHook(() => useBacklogHandlers(opts));

    act(() => {
      result.current.handleTargetToggle("t1");
    });

    expect(useBacklogDetailUIStore.getState().selectedTargetIds.has("t1")).toBe(true);

    act(() => {
      result.current.handleTargetToggle("t1");
    });

    expect(useBacklogDetailUIStore.getState().selectedTargetIds.has("t1")).toBe(false);
  });
});
