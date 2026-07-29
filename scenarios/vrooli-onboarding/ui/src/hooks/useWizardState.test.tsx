// [REQ:REQ-P0-003] Wizard State Management
import { renderHook, act, waitFor } from "@testing-library/react";
import { vi } from "vitest";
import { useWizardState } from "./useWizardState";
import { TOTAL_STEPS } from "../types";

describe("useWizardState", () => {
  beforeEach(() => {
    // Default: no saved progress
    globalThis.fetch = vi.fn().mockResolvedValue({ ok: false, status: 404 });
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
    globalThis.fetch = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({}) });
    const { result } = renderHook(() => useWizardState());

    for (let i = 0; i < TOTAL_STEPS + 2; i++) {
      act(() => result.current.goNext());
    }
    expect(result.current.currentStep).toBe(TOTAL_STEPS - 1);
  });

  it("goPrev decrements step and caps at 0", () => {
    const { result } = renderHook(() => useWizardState());

    // Go forward first
    globalThis.fetch = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({}) });
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

  it("goToStep navigates to valid step", () => {
    const { result } = renderHook(() => useWizardState());

    act(() => result.current.goToStep(2));
    expect(result.current.currentStep).toBe(2);
  });

  it("goToStep ignores out-of-bounds values", () => {
    const { result } = renderHook(() => useWizardState());

    act(() => result.current.goToStep(2));
    expect(result.current.currentStep).toBe(2);

    // Negative
    act(() => result.current.goToStep(-1));
    expect(result.current.currentStep).toBe(2);

    // Too high
    act(() => result.current.goToStep(TOTAL_STEPS));
    expect(result.current.currentStep).toBe(2);

    act(() => result.current.goToStep(100));
    expect(result.current.currentStep).toBe(2);
  });

  it("toggleScenario commits enabled choices to operator state", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({ version: "1.0.0", updated_at: "now", scenarios: {} }) });
    const { result } = renderHook(() => useWizardState());

    act(() => result.current.toggleScenario("scenario-a"));
    expect(result.current.selectedScenarios.has("scenario-a")).toBe(true);
    await waitFor(() => expect(globalThis.fetch).toHaveBeenCalledTimes(2));
  });

  it("startOver resets navigation and local selections", () => {
    const { result } = renderHook(() => useWizardState());

    // Set up some state
    globalThis.fetch = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({}) });
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

  it("nextLabel changes based on current step", () => {
    const { result } = renderHook(() => useWizardState());

    expect(result.current.nextLabel).toBe("Get Started");

    globalThis.fetch = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({}) });
    act(() => result.current.goNext());
    expect(result.current.nextLabel).toBe("Next");

    act(() => result.current.goNext());
    expect(result.current.nextLabel).toBe("Next");
  });

  it("uses a truthful label before validation", () => {
    const { result } = renderHook(() => useWizardState());
    act(() => result.current.goToStep(TOTAL_STEPS - 2));
    expect(result.current.nextLabel).toBe("Review readiness");
  });

  it("isLastStep is true only on the final step", () => {
    const { result } = renderHook(() => useWizardState());
    expect(result.current.isLastStep).toBe(false);

    globalThis.fetch = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({}) });
    for (let i = 0; i < TOTAL_STEPS - 1; i++) {
      act(() => result.current.goNext());
    }
    expect(result.current.isLastStep).toBe(true);
  });

  it("loads selected scenarios from operator state on mount", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          version: "1.0.0",
          updated_at: "2026-07-29T00:00:00Z",
          scenarios: { "scenario-a": { enabled: true }, "scenario-b": { enabled: false } },
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
          version: "1.0.0", updated_at: "2026-07-29T00:00:00Z", scenarios: { alpha: { enabled: true } },
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
          version: "1.0.0", updated_at: "2026-07-29T00:00:00Z", scenarios: { bad: {} },
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
    const rafSpy = vi.spyOn(window, "requestAnimationFrame").mockImplementation((cb) => {
      cb(0);
      return 0;
    });

    // Render hook inside a wrapper that provides a real h1 inside stepContentRef
    const Wrapper = ({ children }: { children: React.ReactNode }) => <>{children}</>;
    const { result } = renderHook(() => useWizardState(), { wrapper: Wrapper });

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

    globalThis.fetch = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({}) });
    act(() => result.current.goNext());

    expect(rafSpy).toHaveBeenCalled();
    expect(heading.getAttribute("tabindex")).toBe("-1");
    expect(focusSpy).toHaveBeenCalled();

    document.body.removeChild(container);
    rafSpy.mockRestore();
  });
});
