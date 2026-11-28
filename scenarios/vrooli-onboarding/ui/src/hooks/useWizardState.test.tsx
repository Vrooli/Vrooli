// [REQ:REQ-P0-003] Wizard State Management
import { renderHook as renderHookBase, act, waitFor } from "../test-utils";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";
import { vi } from "vitest";

vi.mock("../api/session", () => ({
  advanceSessionStep: vi.fn(),
  fetchProfileSession: vi.fn(),
  fetchSession: vi.fn(),
  fetchStepModel: vi.fn(),
  saveProfileSession: vi.fn(),
}));
vi.mock("../api/operatorstate", () => ({
  fetchOperatorState: vi.fn(),
  saveOperatorStateAtRevision: vi.fn(),
}));
vi.mock("../api/selection", () => ({
  acceptRecommendation: vi.fn(),
}));
vi.mock("../api/host", () => ({
  writeHostSafeguardPermission: vi.fn().mockResolvedValue({}),
  writeHostToolPermission: vi.fn().mockResolvedValue({}),
}));
vi.mock("../api/coreAuthority", () => ({
  readCoreAuthority: vi.fn().mockResolvedValue({ revision: "core-r1", authority: { seed: [], trusted_base: [], important: [] } }),
  writeCoreAuthority: vi.fn().mockResolvedValue({ id: "receipt", resultRevision: "core-r2" }),
}));

import { useWizardState } from "./useWizardState";
import { advanceSessionStep, fetchProfileSession, fetchSession, fetchStepModel, saveProfileSession } from "../api/session";
import { fetchOperatorState, saveOperatorStateAtRevision } from "../api/operatorstate";
import { acceptRecommendation as acceptRecommendationRequest } from "../api/selection";
import { writeHostSafeguardPermission, writeHostToolPermission } from "../api/host";
import { readCoreAuthority, writeCoreAuthority } from "../api/coreAuthority";

function renderHook<T>(callback: () => T, options?: Parameters<typeof renderHookBase>[1]) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const inner = options?.wrapper;
  const Wrapper = ({ children }: { children: React.ReactNode }) => {
    const content = inner ? React.createElement(inner, null, children) : children;
    return <QueryClientProvider client={client}>{content}</QueryClientProvider>;
  };
  return renderHookBase(callback, { ...options, wrapper: Wrapper });
}
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
    vi.mocked(fetchProfileSession).mockResolvedValue(null);
    vi.mocked(fetchSession).mockResolvedValue(create(GetSessionResponseSchema, { firstUnsatisfiedStep: 0, completion: false }) as unknown as Awaited<ReturnType<typeof fetchSession>>);
    vi.mocked(advanceSessionStep).mockResolvedValue(create(GetSessionResponseSchema, { firstUnsatisfiedStep: 0, completion: false }) as unknown as Awaited<ReturnType<typeof advanceSessionStep>>);
    vi.mocked(saveProfileSession).mockResolvedValue({ target: "local", actor: "test", mode: "guided", baseRevision: "now", answers: {}, manualDecisions: {}, targetContext: {}, revision: "saved" });
    vi.mocked(fetchOperatorState).mockResolvedValue(testAPIResponse);
    vi.mocked(saveOperatorStateAtRevision).mockResolvedValue(testAPIResponse);
    vi.mocked(acceptRecommendationRequest).mockResolvedValue({ firstUnsatisfiedStep: 2 } as never);
    vi.mocked(writeHostSafeguardPermission).mockResolvedValue({} as never);
    vi.mocked(writeHostToolPermission).mockResolvedValue({} as never);
    vi.mocked(readCoreAuthority).mockResolvedValue({ revision: "core-r1", authority: { seed: [], trusted_base: [], important: [] } });
    vi.mocked(writeCoreAuthority).mockResolvedValue({ id: "receipt", expectedRevision: "core-r1", resultRevision: "core-r2", idempotencyKey: "request", outcome: "applied" });
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
        "local",
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
      "local",
    );
    expect(result.current.operatorStateSaveState).toBe("saved");
  });

  it("restores a persisted scenario after reload and rebases a revision conflict against the latest owner state", async () => {
    const persistedAfterRestart = {
      ...testAPIResponse,
      version: "1.3.0",
      updatedAt: "",
      scenarios: { "hello-python": { enabled: true } },
    };
    vi.mocked(fetchOperatorState).mockResolvedValueOnce(persistedAfterRestart);
    vi.mocked(saveOperatorStateAtRevision).mockRejectedValueOnce(new Error("revision conflict"));

    const { result } = renderHook(() => useWizardState());
    await waitForStepModel(result);
    await waitFor(() => expect(result.current.selectedScenarios.has("hello-python")).toBe(true));

    // The restarted service has restored the durable selection. A concurrent
    // owner write advances its revision after this client reads it.
    act(() => result.current.toggleScenario("hello-python"));
    await waitFor(() => expect(result.current.operatorStateSaveState).toBe("conflict"));
    expect(saveOperatorStateAtRevision).toHaveBeenLastCalledWith(
      { scenarios: { "hello-python": { enabled: false } } },
      "1.3.0",
      "local",
    );

    const latestOwnerState = {
      ...persistedAfterRestart,
      updatedAt: "2026-11-22T19:00:00Z",
      scenarios: {
        "hello-python": { enabled: true },
        "other-client-choice": { enabled: true },
      },
    };
    vi.mocked(fetchOperatorState).mockResolvedValueOnce(latestOwnerState);
    vi.mocked(saveOperatorStateAtRevision).mockResolvedValueOnce({
      ...latestOwnerState,
      updatedAt: "2026-11-22T19:00:01Z",
      scenarios: {
        "hello-python": { enabled: false },
        "other-client-choice": { enabled: true },
      },
    });

    await act(async () => { await result.current.retryOperatorStateSave(); });
    expect(saveOperatorStateAtRevision).toHaveBeenLastCalledWith(
      { scenarios: { "hello-python": { enabled: false } } },
      "2026-11-22T19:00:00Z",
      "local",
    );
    expect(result.current.selectedScenarios.has("hello-python")).toBe(false);
    expect(result.current.selectedScenarios.has("other-client-choice")).toBe(true);
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
    await waitFor(() => expect(saveOperatorStateAtRevision).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(writeCoreAuthority).toHaveBeenCalledWith(
      { seed: ["alpha", "zeta"], trusted_base: [], important: [], members: [] }, "core-r1", expect.any(String), "local",
    ));
    await waitFor(() => expect(writeHostSafeguardPermission).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(writeHostToolPermission).toHaveBeenCalledWith("git", true, "local"));
    expect(writeHostSafeguardPermission).toHaveBeenNthCalledWith(1, "safe", { opted_in: true }, "local");
    expect(writeHostSafeguardPermission).toHaveBeenNthCalledWith(2, "safe", { config: { mode: "guard" } }, "local");
    const saveMock = vi.mocked(saveOperatorStateAtRevision);
    const firstPatch = saveMock.mock.calls[0]?.[0];
    const finalPatch = saveMock.mock.calls[1]?.[0];
    expect(firstPatch).toMatchObject({ scenarios: { "scenario-a": { autoRestart: true } } });
    expect(finalPatch).toMatchObject({
      resources: { postgres: { enabled: true } },
    });
  });

  it("retries an ambiguous core write with the same idempotency key and reads back OperatorState", async () => {
    vi.mocked(writeCoreAuthority).mockRejectedValueOnce(new Error("network unavailable"));
    const { result } = renderHook(() => useWizardState());
    await waitForStepModel(result);
    act(() => result.current.setCoreSeed(["beta", "alpha", "alpha"]));
    await waitFor(() => expect(result.current.operatorStateSaveState).toBe("failed"));
    const [authority, revision, key, target] = vi.mocked(writeCoreAuthority).mock.calls[0]!;
    vi.mocked(fetchOperatorState).mockResolvedValueOnce({ ...testAPIResponse, core: { seed: ["alpha", "beta"], trustedBase: ["alpha"] } });
    await act(async () => { await result.current.retryOperatorStateSave(); });
    expect(writeCoreAuthority).toHaveBeenLastCalledWith(authority, revision, key, target);
    expect(fetchOperatorState).toHaveBeenCalledTimes(2);
    expect(result.current.operatorState?.core).toEqual({ seed: ["alpha", "beta"], trustedBase: ["alpha"] });
    expect(result.current.operatorStateSaveState).toBe("saved");
  });

  it("saves and retries first profile answers against the default schema revision", async () => {
    vi.mocked(fetchOperatorState).mockResolvedValue({ ...testAPIResponse, updatedAt: "", version: "1.0.0" });
    vi.mocked(saveProfileSession)
      .mockRejectedValueOnce(new Error("temporary network failure"))
      .mockResolvedValueOnce({
        target: "local", actor: "test", mode: "guided", profileId: "local-use",
        baseRevision: "1.0.0", answers: { purpose: "use-local-apps" }, manualDecisions: {},
        targetContext: {}, revision: "2026-11-22T12:00:00Z",
      });
    const { result } = renderHook(() => useWizardState());
    await waitForStepModel(result);
    const draft = {
      target: "local", mode: "guided" as const, profileId: "local-use",
      baseRevision: "", answers: { purpose: "use-local-apps" }, manualDecisions: {}, targetContext: {},
    };

    act(() => result.current.persistProfileSession(draft));
    await waitFor(() => expect(result.current.profileSessionSaveState).toBe("failed"));
    expect(result.current.profileSessionError).toMatch(/remain in this tab but are not saved/i);
    expect(vi.mocked(saveProfileSession).mock.calls[0]?.[0]).toMatchObject({
      baseRevision: "1.0.0", expectedRevision: "1.0.0", answers: { purpose: "use-local-apps" },
    });

    await act(async () => { await result.current.retryProfileSessionSave(); });
    expect(vi.mocked(saveProfileSession)).toHaveBeenCalledTimes(2);
    expect(vi.mocked(saveProfileSession).mock.calls[1]?.[0]).toMatchObject({
      baseRevision: "1.0.0", expectedRevision: "1.0.0", answers: { purpose: "use-local-apps" },
    });
    expect(result.current.profileSession?.answers).toEqual({ purpose: "use-local-apps" });
    expect(result.current.profileSessionSaveState).toBe("saved");
  });

  it("retains the core request identity when the initial permission read fails", async () => {
    vi.mocked(readCoreAuthority).mockRejectedValueOnce(new Error("read unavailable"));
    const { result } = renderHook(() => useWizardState());
    await waitForStepModel(result);
    act(() => result.current.setCoreSeed(["beta", "alpha"]));
    await waitFor(() => expect(result.current.operatorStateSaveState).toBe("failed"));
    expect(result.current.operatorState?.core?.seed).toEqual(["alpha", "beta"]);
    expect(writeCoreAuthority).not.toHaveBeenCalled();

    await act(async () => { await result.current.retryOperatorStateSave(); });
    expect(readCoreAuthority).toHaveBeenCalledTimes(3);
    expect(writeCoreAuthority).toHaveBeenCalledWith(
      { seed: ["alpha", "beta"], trusted_base: [], important: [], members: [] }, "core-r1", expect.any(String), "local",
    );
    expect(result.current.operatorStateSaveState).toBe("saved");
  });

  it("uses the accepted recommendation step and exposes the plan as accepted", async () => {
    const { result } = renderHook(() => useWizardState());
    await waitForStepModel(result);
    await act(async () => { await result.current.acceptRecommendation("balanced", ["demo"]); });
    expect(acceptRecommendationRequest).toHaveBeenCalledWith("local", "balanced", ["demo"]);
    expect(result.current.planAccepted).toBe(true);
    expect(result.current.currentStep).toBe(2);
  });

  it("honors accepted step zero over a stale session step without rereading the session", async () => {
    vi.mocked(fetchSession).mockResolvedValue(create(GetSessionResponseSchema, { firstUnsatisfiedStep: 4, completion: false }) as unknown as Awaited<ReturnType<typeof fetchSession>>);
    vi.mocked(acceptRecommendationRequest).mockResolvedValue({ firstUnsatisfiedStep: 0 } as never);

    const { result } = renderHook(() => useWizardState());
    await waitForStepModel(result);
    await waitFor(() => expect(result.current.currentStep).toBe(4));
    const sessionReads = vi.mocked(fetchSession).mock.calls.length;

    await act(async () => { await result.current.acceptRecommendation("balanced", ["demo"]); });

    expect(result.current.currentStep).toBe(0);
    expect(fetchSession).toHaveBeenCalledTimes(sessionReads);
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

  it("restores the durable profile session on mount", async () => {
    vi.mocked(fetchProfileSession).mockResolvedValue({
      target: "local",
      actor: "scenario-authenticator:operator",
      mode: "guided",
      profileId: "develop-and-publish",
      profileVersion: "1.0.0",
      baseRevision: "selection-r1",
      answers: { purposes: ["develop-apps"] },
      manualDecisions: {},
      targetContext: { operation: "prepare-desktop-release" },
      revision: "state-r2",
    });

    const { result } = renderHook(() => useWizardState());

    await waitFor(() => expect(result.current.profileSession?.profileId).toBe("develop-and-publish"));
    expect(result.current.profileSession?.answers).toEqual({ purposes: ["develop-apps"] });
    expect(result.current.profileSession?.revision).toBe("state-r2");
  });

  it("retains a failed profile session and retries against the latest revision", async () => {
    vi.mocked(saveProfileSession).mockRejectedValueOnce(new Error("conflict"));
    const { result } = renderHook(() => useWizardState());
    await waitForStepModel(result);

    act(() => result.current.persistProfileSession({
      target: "local",
      mode: "guided",
      profileId: "local-use",
      profileVersion: "1.0.0",
      baseRevision: "selection-r1",
      answers: { purposes: ["use-local-apps"] },
      manualDecisions: {},
      targetContext: {},
    }));

    await waitFor(() => expect(result.current.profileSessionSaveState).toBe("conflict"));
    expect(result.current.profileSessionError).toMatch(/retained|rebase/i);

    await act(async () => { await result.current.retryProfileSessionSave(); });
    expect(result.current.profileSessionSaveState).toBe("saved");
    expect(saveProfileSession).toHaveBeenCalledTimes(2);
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
