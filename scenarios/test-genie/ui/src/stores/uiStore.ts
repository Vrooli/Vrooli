import { create } from "zustand";
import type { DashboardTabKey, ExecutionFormState, RunsSubtabKey, ScenarioDetailTabKey } from "../types";

const dashboardTabSet = new Set(["dashboard", "runs", "docs", "health"]);
const runsSubtabSet = new Set(["scenarios", "history"]);

export const initialExecutionForm: ExecutionFormState = { scenarioName: "", preset: "quick", failFast: true };

interface UIState {
  activeTab: DashboardTabKey;
  runsSubtab: RunsSubtabKey;
  selectedScenario: string | null;
  scenarioDetailTab: ScenarioDetailTabKey;
  focusScenario: string;
  executionForm: ExecutionFormState;
  executionFeedback: string | null;
  scenarioSearch: string;
  docsSearch: string;
  urlHash: string;
  setActiveTab(tab: DashboardTabKey): void;
  setRunsSubtab(tab: RunsSubtabKey): void;
  setSelectedScenario(scenario: string | null): void;
  setScenarioDetailTab(tab: ScenarioDetailTabKey): void;
  setFocusScenario(scenario: string): void;
  clearFocusScenario(): void;
  applyFocusScenario(scenario: string): void;
  setExecutionForm(form: Partial<ExecutionFormState> | ((prev: ExecutionFormState) => ExecutionFormState)): void;
  resetExecutionForm(): void;
  setExecutionFeedback(feedback: string | null): void;
  setScenarioSearch(search: string): void;
  setDocsSearch(search: string): void;
  navigateToScenarioDetail(scenarioName: string): void;
  navigateBack(): void;
  syncFromHash(): void;
  updateHash(): void;
}

function parseHash(hash: string): Partial<Pick<UIState, "activeTab" | "runsSubtab" | "selectedScenario">> {
  const parts = hash.replace(/^#/, "").split("/").filter(Boolean);
  const parsed: Partial<Pick<UIState, "activeTab" | "runsSubtab" | "selectedScenario">> = {};
  if (parts[0] && dashboardTabSet.has(parts[0])) parsed.activeTab = parts[0] as DashboardTabKey;
  if (parts[0] === "runs") { if (parts[1] && runsSubtabSet.has(parts[1])) parsed.runsSubtab = parts[1] as RunsSubtabKey; if (parts[2]) parsed.selectedScenario = decodeURIComponent(parts[2]); }
  return parsed;
}

function buildHash(state: Pick<UIState, "activeTab" | "runsSubtab" | "selectedScenario">): string {
  if (state.activeTab !== "runs") return state.activeTab;
  return ["runs", state.runsSubtab, state.selectedScenario ? encodeURIComponent(state.selectedScenario) : ""].filter(Boolean).join("/");
}

export const useUIStore = create<UIState>((set, get) => ({
  activeTab: "dashboard", runsSubtab: "scenarios", selectedScenario: null, scenarioDetailTab: "overview", focusScenario: "", executionForm: initialExecutionForm, executionFeedback: null, scenarioSearch: "", docsSearch: "", urlHash: "",
  setActiveTab: (activeTab) => { set({ activeTab, selectedScenario: null }); get().updateHash(); },
  setRunsSubtab: (runsSubtab) => { set({ runsSubtab, selectedScenario: null }); get().updateHash(); },
  setSelectedScenario: (selectedScenario) => { set({ selectedScenario, scenarioDetailTab: "overview" }); get().updateHash(); },
  setScenarioDetailTab: (scenarioDetailTab) => set({ scenarioDetailTab }),
  setFocusScenario: (focusScenario) => set({ focusScenario }),
  clearFocusScenario: () => set({ focusScenario: "", executionForm: { ...get().executionForm, scenarioName: "" } }),
  applyFocusScenario: (scenario) => { const focusScenario = scenario.trim(); set({ focusScenario, executionForm: { ...get().executionForm, scenarioName: focusScenario } }); },
  setExecutionForm: (form) => set((state) => ({ executionForm: typeof form === "function" ? form(state.executionForm) : { ...state.executionForm, ...form } })),
  resetExecutionForm: () => set({ executionForm: initialExecutionForm }),
  setExecutionFeedback: (executionFeedback) => set({ executionFeedback }),
  setScenarioSearch: (scenarioSearch) => set({ scenarioSearch }),
  setDocsSearch: (docsSearch) => set({ docsSearch }),
  navigateToScenarioDetail: (selectedScenario) => { set({ activeTab: "runs", runsSubtab: "scenarios", selectedScenario }); get().updateHash(); },
  navigateBack: () => { set({ selectedScenario: null, scenarioDetailTab: "overview" }); get().updateHash(); },
  syncFromHash: () => { if (typeof window !== "undefined") set(parseHash(window.location.hash)); },
  updateHash: () => { if (typeof window === "undefined") return; const hash = buildHash(get()); if (hash !== get().urlHash) { window.history.replaceState(null, "", `#${hash}`); set({ urlHash: hash }); } }
}));

if (typeof window !== "undefined") {
  useUIStore.setState(parseHash(window.location.hash));
  window.addEventListener("hashchange", () => useUIStore.getState().syncFromHash());
}
