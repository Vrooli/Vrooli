import { create } from "zustand";
import type { DashboardTabKey, RunsSubtabKey, ScenarioDetailTabKey, QueueFormState, ExecutionFormState } from "../types";

const DEFAULT_REQUEST_TYPES = ["unit", "integration"];

const initialQueueForm: QueueFormState = {
  scenarioName: "",
  requestedTypes: DEFAULT_REQUEST_TYPES,
  coverageTarget: 95,
  priority: "normal",
  notes: ""
};

const initialExecutionForm: ExecutionFormState = {
  scenarioName: "",
  preset: "quick",
  failFast: true,
  suiteRequestId: ""
};

interface UIState {
  // Navigation state
  activeTab: DashboardTabKey;
  runsSubtab: RunsSubtabKey;
  selectedScenario: string | null;
  scenarioDetailTab: ScenarioDetailTabKey;

  // Focus state
  focusScenario: string;

  // Form states
  queueForm: QueueFormState;
  executionForm: ExecutionFormState;

  // Feedback
  queueFeedback: string | null;
  executionFeedback: string | null;

  // Search states
  scenarioSearch: string;
  docsSearch: string;

  // URL hash sync
  urlHash: string;

  // Actions
  setActiveTab: (tab: DashboardTabKey) => void;
  setRunsSubtab: (subtab: RunsSubtabKey) => void;
  setSelectedScenario: (scenario: string | null) => void;
  setScenarioDetailTab: (tab: ScenarioDetailTabKey) => void;
  setFocusScenario: (scenario: string) => void;
  clearFocusScenario: () => void;
  applyFocusScenario: (scenario: string) => void;

  setQueueForm: (form: Partial<QueueFormState> | ((prev: QueueFormState) => QueueFormState)) => void;
  setExecutionForm: (form: Partial<ExecutionFormState> | ((prev: ExecutionFormState) => ExecutionFormState)) => void;
  resetQueueForm: () => void;
  resetExecutionForm: () => void;
  toggleRequestedType: (type: string) => void;

  setQueueFeedback: (feedback: string | null) => void;
  setExecutionFeedback: (feedback: string | null) => void;

  setScenarioSearch: (search: string) => void;
  setDocsSearch: (search: string) => void;

  // Navigation helpers
  navigateToScenarioDetail: (scenarioName: string) => void;
  navigateBack: () => void;
  syncFromHash: () => void;
  updateHash: () => void;
}

// Parse URL hash to extract state
function parseHash(hash: string): Partial<Pick<UIState, "activeTab" | "runsSubtab" | "selectedScenario">> {
  const cleanHash = hash.replace(/^#/, "");
  const parts = cleanHash.split("/").filter(Boolean);

  const result: Partial<Pick<UIState, "activeTab" | "runsSubtab" | "selectedScenario">> = {};

  if (parts.length > 0) {
    const tab = parts[0];
    if (["dashboard", "runs", "generate", "docs"].includes(tab)) {
      result.activeTab = tab as DashboardTabKey;
    }
  }

  if (parts.length > 1 && parts[0] === "runs") {
    const subtab = parts[1];
    if (["scenarios", "history"].includes(subtab)) {
      result.runsSubtab = subtab as RunsSubtabKey;
    }
    if (parts.length > 2) {
      result.selectedScenario = decodeURIComponent(parts[2]);
    }
  }

  return result;
}

// Build URL hash from state
function buildHash(state: Pick<UIState, "activeTab" | "runsSubtab" | "selectedScenario">): string {
  const parts: string[] = [state.activeTab];

  if (state.activeTab === "runs") {
    parts.push(state.runsSubtab);
    if (state.selectedScenario) {
      parts.push(encodeURIComponent(state.selectedScenario));
    }
  }

  return parts.join("/");
}

export const useUIStore = create<UIState>((set, get) => ({
  // Initial state
  activeTab: "dashboard",
  runsSubtab: "scenarios",
  selectedScenario: null,
  scenarioDetailTab: "overview",
  focusScenario: "",
  queueForm: initialQueueForm,
  executionForm: initialExecutionForm,
  queueFeedback: null,
  executionFeedback: null,
  scenarioSearch: "",
  docsSearch: "",
  urlHash: "",

  // Navigation actions
  setActiveTab: (tab) => {
    set({ activeTab: tab, selectedScenario: null });
    get().updateHash();
  },

  setRunsSubtab: (subtab) => {
    set({ runsSubtab: subtab, selectedScenario: null });
    get().updateHash();
  },

  setSelectedScenario: (scenario) => {
    set({ selectedScenario: scenario, scenarioDetailTab: "overview" });
    get().updateHash();
  },

  setScenarioDetailTab: (tab) => {
    set({ scenarioDetailTab: tab });
  },

  // Focus actions
  setFocusScenario: (scenario) => set({ focusScenario: scenario }),

  clearFocusScenario: () => {
    set({
      focusScenario: "",
      queueForm: { ...get().queueForm, scenarioName: "" },
      executionForm: { ...get().executionForm, scenarioName: "" }
    });
  },

  applyFocusScenario: (scenario) => {
    const trimmed = scenario.trim();
    set({
      focusScenario: trimmed,
      queueForm: { ...get().queueForm, scenarioName: trimmed },
      executionForm: { ...get().executionForm, scenarioName: trimmed }
    });
  },

  // Form actions
  setQueueForm: (form) => {
    if (typeof form === "function") {
      set((state) => ({ queueForm: form(state.queueForm) }));
    } else {
      set((state) => ({ queueForm: { ...state.queueForm, ...form } }));
    }
  },

  setExecutionForm: (form) => {
    if (typeof form === "function") {
      set((state) => ({ executionForm: form(state.executionForm) }));
    } else {
      set((state) => ({ executionForm: { ...state.executionForm, ...form } }));
    }
  },

  resetQueueForm: () => set({ queueForm: initialQueueForm }),
  resetExecutionForm: () => set({ executionForm: initialExecutionForm }),

  toggleRequestedType: (type) => {
    set((state) => {
      const exists = state.queueForm.requestedTypes.includes(type);
      const nextTypes = exists
        ? state.queueForm.requestedTypes.filter((t) => t !== type)
        : [...state.queueForm.requestedTypes, type];
      return { queueForm: { ...state.queueForm, requestedTypes: nextTypes } };
    });
  },

  // Feedback actions
  setQueueFeedback: (feedback) => set({ queueFeedback: feedback }),
  setExecutionFeedback: (feedback) => set({ executionFeedback: feedback }),

  // Search actions
  setScenarioSearch: (search) => set({ scenarioSearch: search }),
  setDocsSearch: (search) => set({ docsSearch: search }),

  // Navigation helpers
  navigateToScenarioDetail: (scenarioName) => {
    set({
      activeTab: "runs",
      runsSubtab: "scenarios",
      selectedScenario: scenarioName
    });
    get().updateHash();
  },

  navigateBack: () => {
    set({ selectedScenario: null, scenarioDetailTab: "overview" });
    get().updateHash();
  },

  syncFromHash: () => {
    if (typeof window === "undefined") return;
    const parsed = parseHash(window.location.hash);
    set(parsed);
  },

  updateHash: () => {
    if (typeof window === "undefined") return;
    const state = get();
    const hash = buildHash(state);
    if (hash !== state.urlHash) {
      window.history.replaceState(null, "", `#${hash}`);
      set({ urlHash: hash });
    }
  }
}));

// Initialize from URL hash on load
if (typeof window !== "undefined") {
  const initialParsed = parseHash(window.location.hash);
  if (Object.keys(initialParsed).length > 0) {
    useUIStore.setState(initialParsed);
  }

  // Listen for hash changes
  window.addEventListener("hashchange", () => {
    useUIStore.getState().syncFromHash();
  });
}

export { DEFAULT_REQUEST_TYPES, initialQueueForm, initialExecutionForm };
