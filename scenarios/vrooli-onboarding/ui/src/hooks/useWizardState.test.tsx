// [REQ:REQ-P0-003] Wizard State Management
import { renderHook, act, waitFor } from "@testing-library/react";
import { vi } from "vitest";
import { useWizardState } from "./useWizardState";

const testSteps = [
  "welcome",
  "scenarios",
  "resources",
  "credentials",
  "integrations",
  "host",
  "operating-mode",
  "apply",
  "validation",
].map((id, ordinal) => ({
  id,
  ordinal,
  title: id,
  route: `/setup/${id}`,
  deferred: false,
}));
const stepCount = testSteps.length;
const testAPIResponse = {
  steps: testSteps,
  version: "1.0.0",
  updated_at: "now",
  scenarios: {},
};

async function waitForStepModel(result: { current: { steps: unknown[] } }) {
  await waitFor(() => expect(result.current.steps).toHaveLength(stepCount));
}

describe("useWizardState", () => {
  beforeEach(() => {
    window.history.replaceState({}, "", "/");
    // Default: no saved progress
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(testAPIResponse),
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("starts at step 0 with no selected scenarios", () => {
    const { result } = renderHook(() => useWizardState());
    expect(result.current.currentStep).toBe(0);
    expect(result.current.selectedScenarios.size).toBe(0);
  });

  it("goNext advances step and caps at last step", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(testAPIResponse),
    });
    const { result } = renderHook(() => useWizardState());
    await waitForStepModel(result);

    for (let i = 0; i < stepCount + 2; i++) {
      act(() => result.current.goNext());
    }
    expect(result.current.currentStep).toBe(stepCount - 1);
  });

  it("goPrev decrements step and caps at 0", async () => {
    const { result } = renderHook(() => useWizardState());
    await waitForStepModel(result);

    // Go forward first
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(testAPIResponse),
    });
    act(() => result.current.goNext());
    act(() => result.current.goNext());
    expect(result.current.currentStep).toBe(2);

    act(() => result.current.goPrev());
    expect(result.current.currentStep).toBe(1);

    // Should not go below 0
    act(() => result.current.goPrev());
    act(() => result.current.goPrev());
    act(() => result.current.goPrev());
    expect(result.current.currentStep).toBe(0);
  });

  it("goToStep navigates to valid step", async () => {
    const { result } = renderHook(() => useWizardState());
    await waitForStepModel(result);

    act(() => result.current.goToStep(2));
    expect(result.current.currentStep).toBe(2);
  });

  it("goToStep ignores out-of-bounds values", async () => {
    const { result } = renderHook(() => useWizardState());
    await waitForStepModel(result);

    act(() => result.current.goToStep(2));
    expect(result.current.currentStep).toBe(2);

    // Negative
    act(() => result.current.goToStep(-1));
    expect(result.current.currentStep).toBe(2);

    // Too high
    act(() => result.current.goToStep(stepCount));
    expect(result.current.currentStep).toBe(2);

    act(() => result.current.goToStep(100));
    expect(result.current.currentStep).toBe(2);
  });

  it("toggleScenario commits enabled choices to operator state", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(testAPIResponse),
    });
    const { result } = renderHook(() => useWizardState());
    await waitForStepModel(result);

    act(() => result.current.toggleScenario("scenario-a"));
    expect(result.current.selectedScenarios.has("scenario-a")).toBe(true);
    await waitFor(() => expect(globalThis.fetch).toHaveBeenCalledTimes(5));
  });

  it("startOver resets navigation and local selections", async () => {
    const { result } = renderHook(() => useWizardState());
    await waitForStepModel(result);

    // Set up some state
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(testAPIResponse),
    });
    act(() => result.current.toggleScenario("scenario-a"));
    act(() => result.current.goNext());
    act(() => result.current.goNext());
    expect(result.current.currentStep).toBe(2);
    expect(result.current.selectedScenarios.size).toBe(1);

    // Start over
    act(() => result.current.startOver());
    expect(result.current.currentStep).toBe(0);
    expect(result.current.selectedScenarios.size).toBe(0);
  });

  it("nextLabel changes based on current step", async () => {
    const { result } = renderHook(() => useWizardState());
    await waitForStepModel(result);

    expect(result.current.nextLabel).toBe("Get Started");

    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(testAPIResponse),
    });
    act(() => result.current.goNext());
    expect(result.current.nextLabel).toBe("Next");

    act(() => result.current.goNext());
    expect(result.current.nextLabel).toBe("Next");
  });

  it("uses a truthful label before validation", async () => {
    const { result } = renderHook(() => useWizardState());
    await waitForStepModel(result);
    act(() => result.current.goToStep(stepCount - 2));
    expect(result.current.nextLabel).toBe("Review readiness");
  });

  it("isLastStep is true only on the final step", async () => {
    const { result } = renderHook(() => useWizardState());
    await waitForStepModel(result);
    expect(result.current.isLastStep).toBe(false);

    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(testAPIResponse),
    });
    for (let i = 0; i < stepCount - 1; i++) {
      act(() => result.current.goNext());
    }
    expect(result.current.isLastStep).toBe(true);
  });

  it("loads selected scenarios from operator state on mount", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          ...testAPIResponse,
          version: "1.0.0",
          updated_at: "2026-07-29T00:00:00Z",
          scenarios: {
            "scenario-a": { enabled: true },
            "scenario-b": { enabled: false },
          },
        }),
    });

    const { result } = renderHook(() => useWizardState());

    await waitFor(() => {
      expect(result.current.selectedScenarios.has("scenario-a")).toBe(true);
    });
    expect(result.current.selectedScenarios.has("scenario-b")).toBe(false);
  });

  it("does not treat operator state as disposable wizard progress", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          ...testAPIResponse,
          version: "1.0.0",
          updated_at: "2026-07-29T00:00:00Z",
          scenarios: { alpha: { enabled: true } },
        }),
    });

    const { result } = renderHook(() => useWizardState());

    await waitFor(() => {
      expect(result.current.selectedScenarios.has("alpha")).toBe(true);
    });
    expect(result.current.currentStep).toBe(0);
  });

  it("ignores unselected or malformed operator state scenarios", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          ...testAPIResponse,
          version: "1.0.0",
          updated_at: "2026-07-29T00:00:00Z",
          scenarios: { bad: {} },
        }),
    });

    const { result } = renderHook(() => useWizardState());

    await waitFor(() => {
      expect(result.current.operatorState).not.toBeNull();
    });
    expect(result.current.selectedScenarios.size).toBe(0);
  });

  it("handles fetch progress failure gracefully", async () => {
    globalThis.fetch = vi.fn().mockRejectedValue(new Error("Network error"));
    const { result } = renderHook(() => useWizardState());

    // Should start fresh without errors
    // Wait a tick for the effect to run
    await act(async () => {});
    expect(result.current.currentStep).toBe(0);
  });

  it("focuses heading on step change via requestAnimationFrame", async () => {
    // Synchronously execute rAF callback so the DOM mutation is visible
    const rafSpy = vi
      .spyOn(window, "requestAnimationFrame")
      .mockImplementation((cb) => {
        cb(0);
        return 0;
      });

    // Render hook inside a wrapper that provides a real h1 inside stepContentRef
    const Wrapper = ({ children }: { children: React.ReactNode }) => (
      <>{children}</>
    );
    const { result } = renderHook(() => useWizardState(), { wrapper: Wrapper });
    await waitForStepModel(result);

    // Attach a DOM element with an h1 to stepContentRef
    const container = document.createElement("div");
    const heading = document.createElement("h1");
    heading.textContent = "Step Title";
    container.appendChild(heading);
    document.body.appendChild(container);

    // Assign the ref
    Object.defineProperty(result.current.stepContentRef, "current", {
      value: container,
      writable: true,
    });

    const focusSpy = vi.spyOn(heading, "focus");

    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(testAPIResponse),
    });
    act(() => result.current.goNext());

    expect(rafSpy).toHaveBeenCalled();
    expect(heading.getAttribute("tabindex")).toBe("-1");
    expect(focusSpy).toHaveBeenCalled();

    document.body.removeChild(container);
    rafSpy.mockRestore();
  });
});
