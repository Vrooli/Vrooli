import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook as renderHookBase, waitFor } from "../test-utils";
import React from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const api = vi.hoisted(() => ({
  fetchDraft: vi.fn(), saveDraft: vi.fn(), discardDraft: vi.fn(), fetchOperatorState: vi.fn(),
}));

vi.mock("../api/session", () => api);
vi.mock("../api/operatorstate", () => ({ fetchOperatorState: api.fetchOperatorState }));

import { useOperatorQuestionDraft } from "./useOperatorQuestionDraft";

function renderHook<T>(callback: () => T) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const Wrapper = ({ children }: { children: React.ReactNode }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  return renderHookBase(callback, { wrapper: Wrapper });
}

describe("operator question drafts", () => {
  const target = `local-${crypto.randomUUID()}`;
  const questions = [
    { id: "region-choice", kind: "choice", defaultValue: "" },
    { id: "api-secret", kind: "secret", defaultValue: "" },
  ];
  let savedDraft: { target: string; actor: string; baseRevision: string; revision: string; stepId: string; choices: Record<string, string> };

  beforeEach(() => {
    vi.clearAllMocks();
    savedDraft = { target, actor: "onboarding-questionnaire", baseRevision: "v1", revision: "", stepId: "credentials", choices: {} };
    api.fetchDraft.mockImplementation(async () => ({ ...savedDraft, choices: { ...savedDraft.choices } }));
    api.fetchOperatorState.mockResolvedValue({ version: "v1" });
    api.saveDraft.mockImplementation(async (request) => {
      savedDraft = { ...request, revision: "draft-r1", stepId: request.stepId ?? "" };
      return { ...savedDraft };
    });
    api.discardDraft.mockResolvedValue({});
  });

  it("saves and restores nonsecret answers while omitting secret fields", async () => {
    const firstRoute = renderHook(() => useOperatorQuestionDraft(target, questions));
    await waitFor(() => expect(firstRoute.result.current.loading).toBe(false));
    act(() => {
      firstRoute.result.current.store.setValue("region-choice", "eu-west");
      firstRoute.result.current.store.setValue("api-secret", "synthetic-secret-value");
    });
    await waitFor(() => expect(api.saveDraft).toHaveBeenCalledTimes(1), { timeout: 2500 });

    const savedChoices = api.saveDraft.mock.calls[0]?.[0]?.choices as Record<string, string>;
    expect(Object.values(savedChoices)).toContain("eu-west");
    expect(JSON.stringify(savedChoices)).not.toContain("synthetic-secret-value");
    expect(Object.keys(savedChoices).some((key) => key.includes("api-secret"))).toBe(false);
    firstRoute.unmount();

    const resumedRoute = renderHook(() => useOperatorQuestionDraft(target, questions));
    await waitFor(() => expect(resumedRoute.result.current.loading).toBe(false));
    expect(resumedRoute.result.current.store.getValues()["region-choice"]).toBe("eu-west");
    expect(resumedRoute.result.current.store.getValues()["api-secret"]).toBe("");
    resumedRoute.unmount();
  });
});
