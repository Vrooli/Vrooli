import { create } from "zustand";
import {
  listCaptures,
  getCapturesSummary,
  deleteCapture as apiDeleteCapture,
  deleteAllCaptures as apiDeleteAll,
  buildCapturesDownloadUrl,
} from "../lib/api/captures";
import type {
  EvidenceCapture,
  EvidenceCapturesSummary,
} from "@vrooli/proto-types/scenario-to-desktop/v1/domain/evidence_pb";

interface CapturesState {
  isOpen: boolean;
  scenarioName: string | null;
  captures: EvidenceCapture[];
  summary: EvidenceCapturesSummary | null;
  selectedIds: Set<string>;
  loading: boolean;
  error: string | null;
}

interface CapturesActions {
  open: (scenarioName: string) => void;
  close: () => void;
  fetchCaptures: () => Promise<void>;
  fetchSummary: (scenarioName: string) => Promise<void>;
  toggleSelect: (id: string) => void;
  selectAll: () => void;
  deselectAll: () => void;
  deleteCapture: (id: string) => Promise<void>;
  deleteAll: () => Promise<void>;
  downloadSelected: () => void;
}

export type CapturesStore = CapturesState & CapturesActions;

export const useCapturesStore = create<CapturesStore>((set, get) => ({
  isOpen: false,
  scenarioName: null,
  captures: [],
  summary: null,
  selectedIds: new Set(),
  loading: false,
  error: null,

  open: (scenarioName) => {
    set({
      isOpen: true,
      scenarioName,
      captures: [],
      selectedIds: new Set(),
      error: null,
      loading: true,
    });
    void get().fetchCaptures();
  },

  close: () => {
    set({
      isOpen: false,
      scenarioName: null,
      captures: [],
      summary: null,
      selectedIds: new Set(),
      loading: false,
      error: null,
    });
  },

  fetchCaptures: async () => {
    const { scenarioName } = get();
    if (!scenarioName) return;
    set({ loading: true, error: null });
    try {
      const [caps, summary] = await Promise.all([
        listCaptures(scenarioName),
        getCapturesSummary(scenarioName),
      ]);
      set({ captures: caps, summary, loading: false });
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : "Failed to load captures",
        loading: false,
      });
    }
  },

  fetchSummary: async (scenarioName) => {
    try {
      const summary = await getCapturesSummary(scenarioName);
      set({ summary });
    } catch {
      // Best effort
    }
  },

  toggleSelect: (id) => {
    const { selectedIds } = get();
    const next = new Set(selectedIds);
    if (next.has(id)) {
      next.delete(id);
    } else {
      next.add(id);
    }
    set({ selectedIds: next });
  },

  selectAll: () => {
    const ids = new Set(get().captures.map((capture) => capture.captureId));
    set({ selectedIds: ids });
  },

  deselectAll: () => {
    set({ selectedIds: new Set() });
  },

  deleteCapture: async (id) => {
    const { scenarioName } = get();
    if (!scenarioName) return;
    await apiDeleteCapture(scenarioName, id);
    const { selectedIds } = get();
    const next = new Set(selectedIds);
    next.delete(id);
    set({ selectedIds: next });
    await get().fetchCaptures();
  },

  deleteAll: async () => {
    const { scenarioName } = get();
    if (!scenarioName) return;
    await apiDeleteAll(scenarioName);
    set({ selectedIds: new Set() });
    await get().fetchCaptures();
  },

  downloadSelected: () => {
    const { scenarioName, selectedIds } = get();
    if (!scenarioName || selectedIds.size === 0) return;
    const url = buildCapturesDownloadUrl(scenarioName, [...selectedIds]);
    window.open(url, "_blank");
  },
}));
