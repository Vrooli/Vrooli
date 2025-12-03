import { describe, it, expect, beforeEach, vi } from "vitest";
import { useUIStore, DEFAULT_REQUEST_TYPES, initialQueueForm, initialExecutionForm } from "./uiStore";

// Mock window for hash-related tests
const mockWindow = {
  location: { hash: "" },
  history: { replaceState: vi.fn() },
  addEventListener: vi.fn()
};

describe("uiStore", () => {
  beforeEach(() => {
    // Reset store to initial state
    useUIStore.setState({
      activeTab: "dashboard",
      runsSubtab: "scenarios",
      selectedScenario: null,
      focusScenario: "",
      queueForm: initialQueueForm,
      executionForm: initialExecutionForm,
      queueFeedback: null,
      executionFeedback: null,
      scenarioSearch: "",
      docsSearch: "",
      urlHash: ""
    });
  });

  describe("initial state", () => {
    it("has correct initial values", () => {
      const state = useUIStore.getState();
      expect(state.activeTab).toBe("dashboard");
      expect(state.runsSubtab).toBe("scenarios");
      expect(state.selectedScenario).toBeNull();
      expect(state.focusScenario).toBe("");
    });

    it("exports default request types", () => {
      expect(DEFAULT_REQUEST_TYPES).toEqual(["unit", "integration"]);
    });

    it("has correct initial queue form", () => {
      expect(initialQueueForm).toEqual({
        scenarioName: "",
        requestedTypes: ["unit", "integration"],
        coverageTarget: 95,
        priority: "normal",
        notes: ""
      });
    });

    it("has correct initial execution form", () => {
      expect(initialExecutionForm).toEqual({
        scenarioName: "",
        preset: "quick",
        failFast: true,
        suiteRequestId: ""
      });
    });
  });

  describe("navigation actions", () => {
    it("setActiveTab changes the active tab", () => {
      useUIStore.getState().setActiveTab("runs");
      expect(useUIStore.getState().activeTab).toBe("runs");
    });

    it("setActiveTab clears selected scenario", () => {
      useUIStore.setState({ selectedScenario: "test-scenario" });
      useUIStore.getState().setActiveTab("dashboard");
      expect(useUIStore.getState().selectedScenario).toBeNull();
    });

    it("setRunsSubtab changes the subtab", () => {
      useUIStore.getState().setRunsSubtab("history");
      expect(useUIStore.getState().runsSubtab).toBe("history");
    });

    it("setRunsSubtab clears selected scenario", () => {
      useUIStore.setState({ selectedScenario: "test-scenario" });
      useUIStore.getState().setRunsSubtab("history");
      expect(useUIStore.getState().selectedScenario).toBeNull();
    });

    it("setSelectedScenario sets the scenario", () => {
      useUIStore.getState().setSelectedScenario("my-scenario");
      expect(useUIStore.getState().selectedScenario).toBe("my-scenario");
    });
  });

  describe("focus actions", () => {
    it("setFocusScenario sets the focus scenario", () => {
      useUIStore.getState().setFocusScenario("test-scenario");
      expect(useUIStore.getState().focusScenario).toBe("test-scenario");
    });

    it("clearFocusScenario clears focus and form fields", () => {
      useUIStore.setState({
        focusScenario: "test-scenario",
        queueForm: { ...initialQueueForm, scenarioName: "test-scenario" },
        executionForm: { ...initialExecutionForm, scenarioName: "test-scenario" }
      });

      useUIStore.getState().clearFocusScenario();

      expect(useUIStore.getState().focusScenario).toBe("");
      expect(useUIStore.getState().queueForm.scenarioName).toBe("");
      expect(useUIStore.getState().executionForm.scenarioName).toBe("");
    });

    it("applyFocusScenario sets focus and updates forms", () => {
      useUIStore.getState().applyFocusScenario("  my-scenario  ");

      expect(useUIStore.getState().focusScenario).toBe("my-scenario");
      expect(useUIStore.getState().queueForm.scenarioName).toBe("my-scenario");
      expect(useUIStore.getState().executionForm.scenarioName).toBe("my-scenario");
    });
  });

  describe("form actions", () => {
    it("setQueueForm with object merges partial updates", () => {
      useUIStore.getState().setQueueForm({ coverageTarget: 80 });
      const form = useUIStore.getState().queueForm;
      expect(form.coverageTarget).toBe(80);
      expect(form.scenarioName).toBe(""); // Unchanged
    });

    it("setQueueForm with function allows functional updates", () => {
      useUIStore.getState().setQueueForm((prev) => ({
        ...prev,
        coverageTarget: prev.coverageTarget + 5
      }));
      expect(useUIStore.getState().queueForm.coverageTarget).toBe(100);
    });

    it("setExecutionForm with object merges partial updates", () => {
      useUIStore.getState().setExecutionForm({ preset: "comprehensive" });
      const form = useUIStore.getState().executionForm;
      expect(form.preset).toBe("comprehensive");
      expect(form.failFast).toBe(true); // Unchanged
    });

    it("resetQueueForm resets to initial values", () => {
      useUIStore.setState({
        queueForm: { ...initialQueueForm, scenarioName: "changed", coverageTarget: 50 }
      });
      useUIStore.getState().resetQueueForm();
      expect(useUIStore.getState().queueForm).toEqual(initialQueueForm);
    });

    it("resetExecutionForm resets to initial values", () => {
      useUIStore.setState({
        executionForm: { ...initialExecutionForm, preset: "comprehensive" }
      });
      useUIStore.getState().resetExecutionForm();
      expect(useUIStore.getState().executionForm).toEqual(initialExecutionForm);
    });

    it("toggleRequestedType adds type if not present", () => {
      useUIStore.getState().toggleRequestedType("performance");
      expect(useUIStore.getState().queueForm.requestedTypes).toContain("performance");
    });

    it("toggleRequestedType removes type if present", () => {
      useUIStore.getState().toggleRequestedType("unit");
      expect(useUIStore.getState().queueForm.requestedTypes).not.toContain("unit");
    });
  });

  describe("feedback actions", () => {
    it("setQueueFeedback sets feedback message", () => {
      useUIStore.getState().setQueueFeedback("Request submitted");
      expect(useUIStore.getState().queueFeedback).toBe("Request submitted");
    });

    it("setQueueFeedback can clear feedback", () => {
      useUIStore.setState({ queueFeedback: "message" });
      useUIStore.getState().setQueueFeedback(null);
      expect(useUIStore.getState().queueFeedback).toBeNull();
    });

    it("setExecutionFeedback sets feedback message", () => {
      useUIStore.getState().setExecutionFeedback("Execution started");
      expect(useUIStore.getState().executionFeedback).toBe("Execution started");
    });
  });

  describe("search actions", () => {
    it("setScenarioSearch updates search value", () => {
      useUIStore.getState().setScenarioSearch("test");
      expect(useUIStore.getState().scenarioSearch).toBe("test");
    });

    it("setDocsSearch updates docs search value", () => {
      useUIStore.getState().setDocsSearch("installation");
      expect(useUIStore.getState().docsSearch).toBe("installation");
    });
  });

  describe("navigation helpers", () => {
    it("navigateToScenarioDetail sets correct state", () => {
      useUIStore.getState().navigateToScenarioDetail("my-scenario");

      const state = useUIStore.getState();
      expect(state.activeTab).toBe("runs");
      expect(state.runsSubtab).toBe("scenarios");
      expect(state.selectedScenario).toBe("my-scenario");
    });

    it("navigateBack clears selected scenario", () => {
      useUIStore.setState({ selectedScenario: "test-scenario" });
      useUIStore.getState().navigateBack();
      expect(useUIStore.getState().selectedScenario).toBeNull();
    });
  });
});
