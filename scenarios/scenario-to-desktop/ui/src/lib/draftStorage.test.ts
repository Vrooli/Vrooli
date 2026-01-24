/**
 * Tests for draft storage utility functions.
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import {
  loadGeneratorAppState,
  saveGeneratorAppState,
  clearGeneratorAppState,
  type GeneratorAppState,
} from "./draftStorage";

// Mock localStorage
const mockStorage: Record<string, string> = {};

const mockLocalStorage = {
  getItem: vi.fn((key: string) => mockStorage[key] ?? null),
  setItem: vi.fn((key: string, value: string) => {
    mockStorage[key] = value;
  }),
  removeItem: vi.fn((key: string) => {
    delete mockStorage[key];
  }),
  clear: vi.fn(() => {
    Object.keys(mockStorage).forEach((key) => delete mockStorage[key]);
  }),
  length: 0,
  key: vi.fn(),
};

// Setup and teardown
beforeEach(() => {
  // Clear mock storage
  Object.keys(mockStorage).forEach((key) => delete mockStorage[key]);
  vi.clearAllMocks();

  // Mock window.localStorage
  Object.defineProperty(window, "localStorage", {
    value: mockLocalStorage,
    writable: true,
  });
});

// ============================================================================
// loadGeneratorAppState
// ============================================================================

describe("loadGeneratorAppState", () => {
  it("returns null when no stored state exists", () => {
    expect(loadGeneratorAppState()).toBeNull();
  });

  it("returns null for invalid JSON", () => {
    mockStorage["std_generator_app_state_v2"] = "invalid json{";
    expect(loadGeneratorAppState()).toBeNull();
  });

  it("returns null for wrong version", () => {
    const oldVersionState = {
      version: 1,
      viewMode: "wizard",
      selectedScenarioName: "",
    };
    mockStorage["std_generator_app_state_v2"] = JSON.stringify(oldVersionState);
    expect(loadGeneratorAppState()).toBeNull();
  });

  it("returns stored state with correct version", () => {
    const validState: GeneratorAppState = {
      version: 2,
      updatedAt: "2024-01-15T10:00:00Z",
      viewMode: "wizard",
      selectedScenarioName: "my-scenario",
      selectedTemplate: "basic",
      selectionSource: "manual",
      currentBuildId: "build-123",
      installerBuildId: null,
      activeStep: 2,
      userPinnedStep: true,
      docPath: null,
    };
    mockStorage["std_generator_app_state_v2"] = JSON.stringify(validState);

    const result = loadGeneratorAppState();

    expect(result).toEqual(validState);
  });

  it("returns null when localStorage throws", () => {
    mockLocalStorage.getItem.mockImplementationOnce(() => {
      throw new Error("Storage error");
    });

    expect(loadGeneratorAppState()).toBeNull();
  });
});

// ============================================================================
// saveGeneratorAppState
// ============================================================================

describe("saveGeneratorAppState", () => {
  it("saves partial state merged with defaults", () => {
    saveGeneratorAppState({ selectedScenarioName: "test-scenario" });

    expect(mockLocalStorage.setItem).toHaveBeenCalled();
    const savedData = JSON.parse(mockStorage["std_generator_app_state_v2"] ?? "{}");

    expect(savedData.selectedScenarioName).toBe("test-scenario");
    expect(savedData.version).toBe(2);
    expect(savedData.viewMode).toBe("wizard"); // default
    expect(savedData.selectedTemplate).toBe("basic"); // default
  });

  it("updates existing state", () => {
    // First save
    saveGeneratorAppState({
      selectedScenarioName: "first-scenario",
      activeStep: 1,
    });

    // Second save - should merge
    saveGeneratorAppState({
      selectedScenarioName: "second-scenario",
    });

    const savedData = JSON.parse(mockStorage["std_generator_app_state_v2"] ?? "{}");
    expect(savedData.selectedScenarioName).toBe("second-scenario");
    expect(savedData.activeStep).toBe(1); // preserved from first save
  });

  it("always updates version and timestamp", () => {
    const firstTimestamp = new Date().toISOString();
    saveGeneratorAppState({ viewMode: "wizard" });

    const savedData = JSON.parse(mockStorage["std_generator_app_state_v2"] ?? "{}");
    expect(savedData.version).toBe(2);
    expect(savedData.updatedAt).toBeDefined();
    expect(new Date(savedData.updatedAt).getTime()).toBeGreaterThanOrEqual(
      new Date(firstTimestamp).getTime() - 1000
    );
  });

  it("handles all state properties", () => {
    const fullState: Partial<GeneratorAppState> = {
      viewMode: "advanced",
      selectedScenarioName: "full-scenario",
      selectedTemplate: "custom",
      selectionSource: "inventory",
      currentBuildId: "build-456",
      installerBuildId: "installer-789",
      activeStep: 3,
      userPinnedStep: true,
      docPath: "/docs/readme",
    };

    saveGeneratorAppState(fullState);

    const savedData = JSON.parse(mockStorage["std_generator_app_state_v2"] ?? "{}");
    expect(savedData.viewMode).toBe("advanced");
    expect(savedData.selectedTemplate).toBe("custom");
    expect(savedData.selectionSource).toBe("inventory");
    expect(savedData.currentBuildId).toBe("build-456");
    expect(savedData.installerBuildId).toBe("installer-789");
    expect(savedData.activeStep).toBe(3);
    expect(savedData.userPinnedStep).toBe(true);
    expect(savedData.docPath).toBe("/docs/readme");
  });
});

// ============================================================================
// clearGeneratorAppState
// ============================================================================

describe("clearGeneratorAppState", () => {
  it("removes stored state", () => {
    saveGeneratorAppState({ selectedScenarioName: "test" });
    expect(mockStorage["std_generator_app_state_v2"]).toBeDefined();

    clearGeneratorAppState();

    expect(mockLocalStorage.removeItem).toHaveBeenCalledWith("std_generator_app_state_v2");
  });

  it("handles clearing non-existent state", () => {
    clearGeneratorAppState();
    expect(mockLocalStorage.removeItem).toHaveBeenCalled();
  });
});

// ============================================================================
// Edge Cases
// ============================================================================

describe("edge cases", () => {
  it("handles localStorage errors gracefully", () => {
    // Make localStorage throw an error
    mockLocalStorage.getItem.mockImplementationOnce(() => {
      throw new Error("Storage unavailable");
    });

    // Should return null and not throw
    expect(loadGeneratorAppState()).toBeNull();
  });

  it("preserves null values in state", () => {
    saveGeneratorAppState({
      currentBuildId: null,
      installerBuildId: null,
      selectionSource: null,
      docPath: null,
    });

    const savedData = JSON.parse(mockStorage["std_generator_app_state_v2"] ?? "{}");
    expect(savedData.currentBuildId).toBeNull();
    expect(savedData.installerBuildId).toBeNull();
    expect(savedData.selectionSource).toBeNull();
    expect(savedData.docPath).toBeNull();
  });
});
