// [REQ:REQ-P0-003] Wizard State Management
import { renderHook, act, waitFor } from "../test-utils";
import { vi } from "vitest";

vi.mock("../api/session", () => ({
  advanceSessionStep: vi.fn(),
  fetchSession: vi.fn(),
  fetchStepModel: vi.fn(),
}));
vi.mock("../api/operatorstate", () => ({
  fetchOperatorState: vi.fn(),
  saveOperatorStateAtRevision: vi.fn(),
}));
vi.mock("../api/selection", () => ({
  acceptRecommendation: vi.fn(),
}));

import { useWizardState } from "./useWizardState";
import { advanceSessionStep, fetchSession, fetchStepModel } from "../api/session";
import { fetchOperatorState, saveOperatorStateAtRevision } from "../api/operatorstate";
import { acceptRecommendation as acceptRecommendationRequest } from "../api/selection";
import { create } from "@bufbuild/protobuf";
import {
  GetSessionResponseSchema,
  GetStepModelResponseSchema,
  StepSchema,
} from "@vrooli/proto-types/vrooli-onboarding/v1/session/session_pb";

const testSteps = [
  "welcome",
  "scenarios",
  "core-set",
  "resources",
  "credentials",
  "integrations",
  "host",
  "operating-mode",
  "apply",
  "validation",
].map((id, ordinal) => create(StepSchema, {
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
  updatedAt: "now",
  scenarios: {},
};

async function waitForStepModel(result: { current: { steps: unknown[] } }) {
  await waitFor(() => expect(result.current.steps).toHaveLength(stepCount));
}

describe("useWizardState", () => {
  beforeEach(() => {
    window.history.replaceState({}, "", "/");
    vi.mocked(fetchStepModel).mockResolvedValue(create(GetStepModelResponseSchema, { steps: testSteps }) as unknown as Awaited<ReturnType<typeof fetchStepModel>>);
    vi.mocked(fetchSession).mockResolvedValue(create(GetSessionResponseSchema, { firstUnsatisfiedStep: 0, completion: false }) as unknown as Awaited<ReturnType<typeof fetchSession>>);
    vi.mocked(advanceSessionStep).mockResolvedValue(create(GetSessionResponseSchema, { firstUnsatisfiedStep: 0, completion: false }) as unknown as Awaited<ReturnType<typeof advanceSessionStep>>);
    vi.mocked(fetchOperatorState).mockResolvedValue(testAPIResponse);
    vi.mocked(saveOperatorStateAtRevision).mockResolvedValue(testAPIResponse);
    vi.mocked(acceptRecommendationRequest).mockResolvedValue({ firstUnsatisfiedStep: 2 } as never);
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

  it("preserves target and draft context while following a settings deep link", async () => {
    window.history.replaceState({}, "", "/setup/welcome?target=remote-1&setting=setup.scenarios&draft=current");
    const { result } = renderHook(() => useWizardState());
    await waitForStepModel(result);

    act(() => result.current.goNext());

    expect(window.location.pathname).toBe("/setup/scenarios");
    expect(new URLSearchParams(window.location.search).get("target")).toBe("remote-1");
    expect(new URLSearchParams(window.location.search).get("setting")).toBe("setup.scenarios");
    expect(new URLSearchParams(window.location.search).get("draft")).toBe("current");
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
    await waitFor(() => {
      expect(saveOperatorStateAtRevision).toHaveBeenCalledWith(
        { scenarios: { "scenario-a": { enabled: true } } },
        "now",
      );
    });
  });

  it("surfaces an operator-state save failure instead of claiming it persisted", async () => {
    vi.mocked(saveOperatorStateAtRevision).mockRejectedValueOnce(new Error("conflict"));
    const { result } = renderHook(() => useWizardState());
    await waitForStepModel(result);

    act(() => result.current.toggleScenario("scenario-a"));

    await waitFor(() => {
      expect(result.current.operatorStateError).toMatch(/retained|could not be saved/i);
    });
    expect(result.current.operatorStateSaveState).toBe("conflict");
    expect(result.current.selectedScenarios.has("scenario-a")).toBe(true);
  });

  it("rebases a retained failed edit before retrying", async () => {
    vi.mocked(saveOperatorStateAtRevision).mockRejectedValueOnce(new Error("conflict"));
    const { result } = renderHook(() => useWizardState());
    await waitForStepModel(result);
    act(() => result.current.toggleScenario("scenario-a"));
    await waitFor(() => expect(result.current.operatorStateSaveState).toBe("conflict"));

    vi.mocked(fetchOperatorState).mockResolvedValue({ ...testAPIResponse, updatedAt: "server-revision" });
    await act(async () => { await result.current.retryOperatorStateSave(); });
    expect(saveOperatorStateAtRevision).toHaveBeenLastCalledWith(
      { scenarios: { "scenario-a": { enabled: true } } },
      "server-revision",
    );
    expect(result.current.operatorStateSaveState).toBe("saved");
  });

  it("persists each owned choice through the revision-aware writer", async () => {
    const { result } = renderHook(() => useWizardState());
    await waitForStepModel(result);
    act(() => {
      result.current.setScenarioAutoRestart("scenario-a", true);
      result.current.setCoreSeed(["zeta", "alpha", "alpha"]);
      result.current.setHostOptIn("host_tools", "git", true);
      result.current.setHostOptIn("host_safeguards", "safe", true);
      result.current.setHostConfig("host_safeguards", "safe", { mode: "guard" });
      result.current.setResourceEnabled("postgres", true);
    });
    await waitFor(() => expect(saveOperatorStateAtRevision).toHaveBeenCalledTimes(6));
    const saveMock = vi.mocked(saveOperatorStateAtRevision);
    const firstPatch = saveMock.mock.calls[0]?.[0];
    const finalPatch = saveMock.mock.calls[5]?.[0];
    expect(firstPatch).toMatchObject({ scenarios: { "scenario-a": { autoRestart: true } } });
    expect(finalPatch).toMatchObject({
      core: { seed: ["alpha", "zeta"], trustedBase: [] },
      hostTools: { git: { optedIn: true } },
      hostSafeguards: { safe: { config: { mode: "guard" } } },
      resources: { postgres: { enabled: true } },
    });
  });

  it("uses the accepted recommendation step and exposes the plan as accepted", async () => {
    const { result } = renderHook(() => useWizardState());
    await waitForStepModel(result);
    await act(async () => { await result.current.acceptRecommendation("balanced", ["demo"]); });
    expect(acceptRecommendationRequest).toHaveBeenCalledWith("local", "balanced", ["demo"]);
    expect(result.current.planAccepted).toBe(true);
    expect(result.current.currentStep).toBe(2);
  });

  it("keeps a generic save failure retryable when the rebase load also fails", async () => {
    vi.mocked(saveOperatorStateAtRevision).mockRejectedValueOnce(new Error("network unavailable"));
    const { result } = renderHook(() => useWizardState());
    await waitForStepModel(result);
    act(() => result.current.toggleScenario("scenario-a"));
    await waitFor(() => expect(result.current.operatorStateSaveState).toBe("failed"));

    vi.mocked(fetchOperatorState).mockRejectedValueOnce(new Error("still offline"));
    await act(async () => { await result.current.retryOperatorStateSave(); });
    expect(result.current.operatorStateError).toMatch(/remains available/i);
  });

  it("retains the edit when the rebased retry fails again", async () => {
    vi.mocked(saveOperatorStateAtRevision).mockRejectedValueOnce(new Error("network unavailable"));
    const { result } = renderHook(() => useWizardState());
    await waitForStepModel(result);
    act(() => result.current.toggleScenario("scenario-a"));
    await waitFor(() => expect(result.current.operatorStateSaveState).toBe("failed"));

    vi.mocked(fetchOperatorState).mockResolvedValueOnce({ ...testAPIResponse, updatedAt: "server-revision" });
    vi.mocked(saveOperatorStateAtRevision).mockRejectedValueOnce(new Error("retry unavailable"));
    await act(async () => { await result.current.retryOperatorStateSave(); });
    expect(result.current.operatorStateSaveState).toBe("failed");
    expect(result.current.selectedScenarios.has("scenario-a")).toBe(true);
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
    vi.mocked(fetchOperatorState).mockResolvedValue({
      ...testAPIResponse,
      version: "1.0.0",
      updatedAt: "2026-07-29T00:00:00Z",
      scenarios: {
        "scenario-a": { enabled: true },
        "scenario-b": { enabled: false },
      },
    });

    const { result } = renderHook(() => useWizardState());

    await waitFor(() => {
      expect(result.current.selectedScenarios.has("scenario-a")).toBe(true);
    });
    expect(result.current.selectedScenarios.has("scenario-b")).toBe(false);
  });

  it("does not treat operator state as disposable wizard progress", async () => {
    vi.mocked(fetchOperatorState).mockResolvedValue({
      ...testAPIResponse,
      version: "1.0.0",
      updatedAt: "2026-07-29T00:00:00Z",
      scenarios: { alpha: { enabled: true } },
    });

    const { result } = renderHook(() => useWizardState());

    await waitFor(() => {
      expect(result.current.selectedScenarios.has("alpha")).toBe(true);
    });
    expect(result.current.currentStep).toBe(0);
  });

  it("ignores unselected or malformed operator state scenarios", async () => {
    vi.mocked(fetchOperatorState).mockResolvedValue({
      ...testAPIResponse,
      version: "1.0.0",
      updatedAt: "2026-07-29T00:00:00Z",
      scenarios: { bad: {} },
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
