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

  it("starts at step 0 with empty resources", () => {
    const { result } = renderHook(() => useWizardState());
    expect(result.current.currentStep).toBe(0);
    expect(result.current.selectedResources.size).toBe(0);
    expect(result.current.resumeAvailable).toBe(false);
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

  it("toggleResource adds and removes resources", () => {
    const { result } = renderHook(() => useWizardState());

    act(() => result.current.toggleResource("postgres"));
    expect(result.current.selectedResources.has("postgres")).toBe(true);

    act(() => result.current.toggleResource("redis"));
    expect(result.current.selectedResources.has("redis")).toBe(true);
    expect(result.current.selectedResources.size).toBe(2);

    // Toggle off
    act(() => result.current.toggleResource("postgres"));
    expect(result.current.selectedResources.has("postgres")).toBe(false);
    expect(result.current.selectedResources.size).toBe(1);
  });

  it("startOver resets step, resources, and resume state", () => {
    const { result } = renderHook(() => useWizardState());

    // Set up some state
    globalThis.fetch = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({}) });
    act(() => result.current.toggleResource("postgres"));
    act(() => result.current.goNext());
    act(() => result.current.goNext());
    expect(result.current.currentStep).toBe(2);
    expect(result.current.selectedResources.size).toBe(1);

    // Start over
    act(() => result.current.startOver());
    expect(result.current.currentStep).toBe(0);
    expect(result.current.selectedResources.size).toBe(0);
    expect(result.current.resumeAvailable).toBe(false);
  });

  it("nextLabel changes based on current step", () => {
    const { result } = renderHook(() => useWizardState());

    expect(result.current.nextLabel).toBe("Get Started");

    globalThis.fetch = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({}) });
    act(() => result.current.goNext());
    expect(result.current.nextLabel).toBe("Next");

    act(() => result.current.goNext());
    expect(result.current.nextLabel).toBe("Generate Config");
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

  it("loads saved progress on mount and enables resume", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          current_step: 2,
          config_data: { resources: ["postgres", "redis"] },
        }),
    });

    const { result } = renderHook(() => useWizardState());

    await waitFor(() => {
      expect(result.current.resumeAvailable).toBe(true);
    });
    expect(result.current.resumeStep).toBe(2);
    expect(result.current.selectedResources.has("postgres")).toBe(true);
    expect(result.current.selectedResources.has("redis")).toBe(true);
  });

  it("handleResume jumps to saved step and clears resume flag", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          current_step: 3,
          config_data: { resources: ["ollama"] },
        }),
    });

    const { result } = renderHook(() => useWizardState());

    await waitFor(() => {
      expect(result.current.resumeAvailable).toBe(true);
    });

    act(() => result.current.handleResume());
    expect(result.current.currentStep).toBe(3);
    expect(result.current.resumeAvailable).toBe(false);
  });

  it("ignores saved progress with invalid resources data", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          current_step: 1,
          config_data: { resources: [42, null] },
        }),
    });

    const { result } = renderHook(() => useWizardState());

    await waitFor(() => {
      expect(result.current.resumeAvailable).toBe(true);
    });
    // Resources should not be set since they're not strings
    expect(result.current.selectedResources.size).toBe(0);
  });

  it("handles fetch progress failure gracefully", async () => {
    globalThis.fetch = vi.fn().mockRejectedValue(new Error("Network error"));
    const { result } = renderHook(() => useWizardState());

    // Should start fresh without errors
    // Wait a tick for the effect to run
    await act(async () => {});
    expect(result.current.currentStep).toBe(0);
    expect(result.current.resumeAvailable).toBe(false);
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
