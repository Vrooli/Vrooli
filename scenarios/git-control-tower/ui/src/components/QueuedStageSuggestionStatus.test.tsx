import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { StagedMessageJobSchema, StagedMessageMode, StagedMessageProviderReadinessSchema, StagedMessageSuggestionSchema } from "@vrooli/proto-types/git-control-tower/v1/advisory/advisory_pb";
import { renderWithProviders } from "../test-utils/renderWithProviders";
import { getMessageJob, getMessageProviderReadiness, waitMessageJob } from "../features/commit-message/api";
import { QueuedStageSuggestionStatus } from "./QueuedStageSuggestionStatus";
import type { QueuedStageSelection } from "../features/commit-message/stageQueue";

vi.mock("../features/commit-message/api", () => ({ getMessageJob: vi.fn(), getMessageProviderReadiness: vi.fn(), waitMessageJob: vi.fn() }));

const suggestion = create(StagedMessageSuggestionSchema, { title: "fix: preserve queued work", body: "Use the captured snapshot." });
const makeJob = (status = "succeeded", applicable = true) => create(StagedMessageJobSchema, {
  id: "job-1", repositoryId: "repo-1", status, applicable,
  applicabilityReason: applicable ? "" : "The selected files changed.",
  result: suggestion,
});
const selection: QueuedStageSelection = {
  id: "request-1", requestKey: "request-1", jobId: "job-1", paths: ["src/main.ts", "README.md"],
  queuedAt: "2026-11-12T12:00:00.000Z", state: "staged", suggestionMode: "quick",
};

beforeEach(() => {
  vi.mocked(getMessageJob).mockReset().mockResolvedValue(makeJob());
  vi.mocked(getMessageProviderReadiness).mockReset().mockResolvedValue(create(StagedMessageProviderReadinessSchema, { provider: "workspace-sandbox", state: "unknown", known: false, message: "unknown" }));
  vi.mocked(waitMessageJob).mockReset().mockResolvedValue(makeJob("running", true));
});

describe("QueuedSuggestionCard", () => {
  it("reconnects with both persisted handles and waits while the owner is active", async () => {
    let release!: (job: ReturnType<typeof makeJob>) => void;
    vi.mocked(getMessageJob).mockResolvedValue(makeJob("queued"));
    vi.mocked(waitMessageJob).mockImplementation(() => new Promise(resolve => { release = resolve; }));
    renderWithProviders(<QueuedStageSuggestionStatus repositoryId="repo-1" selection={selection} isStaged onUse={vi.fn()} onRetry={vi.fn()} />);

    await waitFor(() => expect(getMessageJob).toHaveBeenCalledWith("repo-1", "job-1", "request-1"));
    await waitFor(() => expect(waitMessageJob).toHaveBeenCalledWith("repo-1", "job-1", "request-1", expect.any(AbortSignal)));
    expect(screen.getByTestId("queued-suggestion-state")).toHaveTextContent("Queued");
    await act(async () => release(makeJob("succeeded")));
    await waitFor(() => expect(screen.getByTestId("queued-suggestion-state")).toHaveTextContent("Succeeded"));
  });

  it("deduplicates duplicate responsive cards and shares intermediate Wait updates", async () => {
    vi.mocked(getMessageJob).mockResolvedValue(makeJob("queued"));
    let waitCalls = 0;
    vi.mocked(waitMessageJob).mockImplementation((_repo, _job, _key, signal) => {
      waitCalls++;
      if (waitCalls === 1) return Promise.resolve(makeJob("running"));
      return new Promise((_resolve, reject) => signal.addEventListener("abort", () => reject(new DOMException("Aborted", "AbortError")), { once: true }));
    });
    const props = { repositoryId: "repo-1", selection, isStaged: true, onUse: vi.fn(), onRetry: vi.fn() };
    renderWithProviders(<><QueuedStageSuggestionStatus {...props} /><QueuedStageSuggestionStatus {...props} /></>);

    await waitFor(() => expect(screen.getAllByTestId("queued-suggestion-state").every(node => node.textContent?.includes("Running"))).toBe(true));
    expect(getMessageJob).toHaveBeenCalledTimes(1);
    expect(waitMessageJob).toHaveBeenCalledTimes(1);
  });

  it.each([
    ["dispatching", "Dispatching"],
    ["awaiting_input", "Awaiting input"],
    ["cancelling", "Cancelling"],
  ])("preserves the distinct %s lifecycle state while waiting", async (status, label) => {
    vi.mocked(getMessageJob).mockResolvedValue(makeJob(status));
    let release!: (job: ReturnType<typeof makeJob>) => void;
    vi.mocked(waitMessageJob).mockImplementation(() => new Promise(resolve => { release = resolve; }));
    renderWithProviders(<QueuedStageSuggestionStatus repositoryId="repo-1" selection={selection} isStaged onUse={vi.fn()} />);
    expect(await screen.findByTestId("queued-suggestion-state")).toHaveTextContent(label);
    await waitFor(() => expect(waitMessageJob).toHaveBeenCalled());
    await act(async () => release(makeJob("succeeded")));
  });

  it("shows cancelled and stale as separate terminal states", async () => {
    vi.mocked(getMessageJob).mockResolvedValue(makeJob("cancelled"));
    const view = renderWithProviders(<QueuedStageSuggestionStatus repositoryId="repo-1" selection={selection} isStaged={false} onUse={vi.fn()} />);
    expect(await screen.findByTestId("queued-suggestion-state")).toHaveTextContent("Cancelled");
    vi.mocked(getMessageJob).mockResolvedValue(makeJob("stale"));
    view.rerender(<QueuedStageSuggestionStatus repositoryId="repo-1" selection={{ ...selection, id: "request-2", requestKey: "request-2" }} isStaged={false} onUse={vi.fn()} />);
    await waitFor(() => expect(screen.getByTestId("queued-suggestion-state")).toHaveTextContent("Stale"));
  });

  it("shows a persisted active-job failure explanation with same-job recovery", async () => {
    const failure = { ...makeJob("running"), errorCode: "evidence_unavailable", error: "Agent Manager finished the execution, but its saved result evidence could not be read. The job remains attached; reconnect to retrieve the same result.", retryable: true };
    vi.mocked(getMessageJob).mockResolvedValue(failure);
    vi.mocked(waitMessageJob).mockImplementation(() => new Promise(() => undefined));
    renderWithProviders(<QueuedStageSuggestionStatus repositoryId="repo-1" selection={selection} isStaged onUse={vi.fn()} />);
    expect(await screen.findByRole("alert")).toHaveTextContent(/saved result evidence could not be read/);
    expect(screen.getByRole("button", { name: "Reconnect to saved job" })).toBeEnabled();
    expect(getMessageJob).toHaveBeenCalledWith("repo-1", "job-1", "request-1");
  });

  it("pauses a new queued-suggestion retry after a maintenance refusal until readiness is open", async () => {
    const failure = { ...makeJob("awaiting_provider"), errorCode: "provider_admission_maintenance", error: "Agent execution is paused for Workspace Sandbox maintenance; no agent work started.", retryable: true, executionStartedKnown: true, executionStarted: false, tokensKnown: true, tokensUsed: 0n, chargeKnown: true, chargeMicroUsd: 0n };
    vi.mocked(getMessageJob).mockResolvedValue(failure);
    const onRefresh = vi.fn();
    const view = renderWithProviders(<QueuedStageSuggestionStatus repositoryId="repo-1" selection={selection} isStaged={false} onUse={vi.fn()} onRefresh={onRefresh} />);
    expect(await screen.findByTestId("queued-suggestion-state")).toHaveTextContent("Waiting for provider admission");
    expect(screen.getByText(/Retry starts a new job only after provider readiness is known open/)).toBeVisible();
    expect(await screen.findByText("Retry is paused until Workspace Sandbox readiness is known open. The saved selection is preserved.")).toBeVisible();
    const retry = screen.getByRole("button", { name: "Start a new job with the same selection" });
    expect(retry).toBeDisabled();
    expect(screen.getByText("Agent did not start · 0 tokens used")).toBeVisible();
    expect(screen.getByText("Marginal charge: $0 USD.")).toBeVisible();
    expect(getMessageProviderReadiness).toHaveBeenCalled();

    view.queryClient.setQueryData(["commit-message", "provider-readiness"], create(StagedMessageProviderReadinessSchema, { provider: "workspace-sandbox", state: "open", known: true, message: "ready" }));
    view.rerender(<QueuedStageSuggestionStatus repositoryId="repo-1" selection={selection} isStaged={false} onUse={vi.fn()} onRefresh={onRefresh} />);
    await waitFor(() => expect(screen.getByRole("button", { name: "Start a new job with the same selection" })).toBeEnabled());
    fireEvent.click(screen.getByRole("button", { name: "Start a new job with the same selection" }));
    expect(onRefresh).toHaveBeenCalledWith(selection);
  });

  it("gates a known pre-dispatch provider outage until readiness is open", async () => {
    const failure = { ...makeJob("failed"), errorCode: "provider_unavailable", error: "Agent Manager message execution is unavailable; no request was dispatched.", retryable: true, executionStartedKnown: true, executionStarted: false, tokensKnown: true, tokensUsed: 0n, chargeKnown: true, chargeMicroUsd: 0n };
    vi.mocked(getMessageJob).mockResolvedValue(failure);
    const onRefresh = vi.fn();
    const view = renderWithProviders(<QueuedStageSuggestionStatus repositoryId="repo-1" selection={selection} isStaged={false} onUse={vi.fn()} onRefresh={onRefresh} />);
    expect(await screen.findByText(/no request was dispatched/)).toBeVisible();
    expect(screen.getByText("Agent did not start · 0 tokens used")).toBeVisible();
    expect(screen.getByText("Marginal charge: $0 USD.")).toBeVisible();
    expect(screen.getByText("Retry is paused until Workspace Sandbox readiness is known open. The saved selection is preserved.")).toBeVisible();
    const retry = screen.getByRole("button", { name: "Start a new job with the same selection" });
    expect(retry).toBeDisabled();
    expect(getMessageProviderReadiness).toHaveBeenCalled();

    view.queryClient.setQueryData(["commit-message", "provider-readiness"], create(StagedMessageProviderReadinessSchema, { provider: "workspace-sandbox", state: "open", known: true, message: "ready" }));
    view.rerender(<QueuedStageSuggestionStatus repositoryId="repo-1" selection={selection} isStaged={false} onUse={vi.fn()} onRefresh={onRefresh} />);
    await waitFor(() => expect(screen.getByRole("button", { name: "Start a new job with the same selection" })).toBeEnabled());
    fireEvent.click(screen.getByRole("button", { name: "Start a new job with the same selection" }));
    expect(onRefresh).toHaveBeenCalledWith(selection);
  });

  it.each([
    [false, 0n, "Marginal charge is unknown; token counts are not a charge estimate."],
    [true, 1_250_000n, "Marginal charge: $1.25 USD."],
  ])("reports charge certainty independently of tokens", async (chargeKnown, amount, expected) => {
    vi.mocked(getMessageJob).mockResolvedValue({ ...makeJob("failed"), chargeKnown, chargeMicroUsd: amount, tokensKnown: true, tokensUsed: 0n });
    renderWithProviders(<QueuedStageSuggestionStatus repositoryId="repo-1" selection={selection} isStaged={false} onUse={vi.fn()} />);
    expect(await screen.findByText(expected)).toBeVisible();
  });

  it("updates elapsed time while owner Wait is still pending", async () => {
    const createdAt = new Date(Date.now() - 1000).toISOString();
    const running = create(StagedMessageJobSchema, { ...makeJob("running"), createdAt, updatedAt: createdAt, phase: "draft" });
    vi.mocked(getMessageJob).mockResolvedValue(running);
    let release!: (job: ReturnType<typeof makeJob>) => void;
    vi.mocked(waitMessageJob).mockImplementation(() => new Promise(resolve => { release = resolve; }));
    renderWithProviders(<QueuedStageSuggestionStatus repositoryId="repo-1" selection={selection} isStaged onUse={vi.fn()} />);
    await waitFor(() => expect(screen.getByText(/Last stage: draft · elapsed 1s/)).toBeVisible());
    await waitFor(() => expect(screen.getByText(/Last stage: draft · elapsed 2s/)).toBeVisible(), { timeout: 3000 });
    await act(async () => release(makeJob("succeeded")));
  });

  it("shows a captured result but offers Use only when the parent stages an applicable job", async () => {
    const onUse = vi.fn();
    const view = renderWithProviders(<QueuedStageSuggestionStatus repositoryId="repo-1" selection={selection} isStaged={false} onUse={onUse} onRetry={vi.fn()} />);
    expect(await screen.findByText("fix: preserve queued work")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Use suggestion" })).not.toBeInTheDocument();

    view.rerender(<QueuedStageSuggestionStatus repositoryId="repo-1" selection={selection} isStaged onUse={onUse} onRetry={vi.fn()} />);
    const button = await screen.findByRole("button", { name: "Use suggestion" });
    button.click();
    expect(onUse).toHaveBeenCalledWith(suggestion);
  });

  it("retains an inapplicable result as stale and explains how to queue the current files", async () => {
    const onRefresh = vi.fn();
    vi.mocked(getMessageJob).mockResolvedValue(makeJob("succeeded", false));
    renderWithProviders(<QueuedStageSuggestionStatus repositoryId="repo-1" selection={selection} isStaged onUse={vi.fn()} onRetry={vi.fn()} onRefresh={onRefresh} />);
    expect(await screen.findByTestId("queued-suggestion-state")).toHaveTextContent("Stale");
    expect(screen.getByText("fix: preserve queued work")).toBeInTheDocument();
    expect(screen.getByText("The selected files changed.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Use suggestion" })).not.toBeInTheDocument();
    expect(screen.getByText(/Remove this selection, then stage and queue the current files as a new selection/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Refresh selection" })).not.toBeInTheDocument();
    expect(onRefresh).not.toHaveBeenCalled();
  });

  it("does not offer a retry for a failed job whose captured files have changed", async () => {
    const onRefresh = vi.fn();
    vi.mocked(getMessageJob).mockResolvedValue(makeJob("failed", false));
    renderWithProviders(<QueuedStageSuggestionStatus repositoryId="repo-1" selection={selection} isStaged={false} onUse={vi.fn()} onRefresh={onRefresh} />);
    expect(await screen.findByTestId("queued-suggestion-state")).toHaveTextContent("Stale");
    expect(screen.queryByRole("button", { name: "Start a new job with the same selection" })).not.toBeInTheDocument();
    expect(onRefresh).not.toHaveBeenCalled();
  });

  it("surfaces a persisted queue failure without attempting a job lookup", () => {
    const failed = { ...selection, jobId: undefined, requestKey: undefined, suggestionState: "failed" as const, suggestionError: "Launch could not be confirmed." };
    renderWithProviders(<QueuedStageSuggestionStatus repositoryId="repo-1" selection={failed} isStaged={false} onUse={vi.fn()} onRetry={vi.fn()} />);
    expect(screen.getByTestId("queued-suggestion-state")).toHaveTextContent("Failed");
    expect(screen.getByText("Launch could not be confirmed.")).toBeInTheDocument();
    expect(getMessageJob).not.toHaveBeenCalled();
  });

  it("keeps a jobless lifecycle refusal behind the provider readiness gate", async () => {
    const onRetry = vi.fn();
    const failed = { ...selection, jobId: undefined, requestKey: undefined, state: "queued" as const, suggestionState: "failed" as const, suggestionError: "Workspace Sandbox maintenance is in progress." };
    const view = renderWithProviders(<QueuedStageSuggestionStatus repositoryId="repo-1" selection={failed} isStaged={false} onUse={vi.fn()} onRetry={onRetry} />);
    const retry = await screen.findByRole("button", { name: "Retry queued suggestion" });
    expect(retry).toBeDisabled();
    expect(getMessageProviderReadiness).toHaveBeenCalled();
    view.queryClient.setQueryData(["commit-message", "provider-readiness"], create(StagedMessageProviderReadinessSchema, { provider: "workspace-sandbox", state: "open", known: true, message: "ready" }));
    view.rerender(<QueuedStageSuggestionStatus repositoryId="repo-1" selection={failed} isStaged={false} onUse={vi.fn()} onRetry={onRetry} />);
    await waitFor(() => expect(screen.getByRole("button", { name: "Retry queued suggestion" })).toBeEnabled());
    fireEvent.click(screen.getByRole("button", { name: "Retry queued suggestion" }));
    expect(onRetry).toHaveBeenCalledWith(failed);
  });

  it("offers a new suggestion for a durable terminal failure, not same-key retry", async () => {
    const onRetry = vi.fn();
    const onRefresh = vi.fn();
    vi.mocked(getMessageJob).mockResolvedValue({ ...makeJob("failed"), retryable: true });
    renderWithProviders(<QueuedStageSuggestionStatus repositoryId="repo-1" selection={selection} isStaged={false} onUse={vi.fn()} onRetry={onRetry} onRefresh={onRefresh} />);

    fireEvent.click(await screen.findByRole("button", { name: "Start a new job with the same selection" }));
    expect(onRefresh).toHaveBeenCalledWith(selection);
    expect(onRetry).not.toHaveBeenCalled();
  });

  it("offers a new job with changed mode, but not a repeat of the non-retryable saved mode", async () => {
    const onRefresh = vi.fn();
    const onStartSuggestion = vi.fn();
    vi.mocked(getMessageJob).mockResolvedValue({ ...makeJob("failed"), errorCode: "budget_exhausted", error: "Drafting stopped because the workflow token budget was exhausted.", retryable: false, tokensKnown: true, tokensUsed: 1200n, chargeKnown: true, chargeMicroUsd: 25000n });
    renderWithProviders(<QueuedStageSuggestionStatus repositoryId="repo-1" selection={selection} isStaged={false} onUse={vi.fn()} onRefresh={onRefresh} onStartSuggestion={onStartSuggestion} />);
    expect(await screen.findByText("Drafting stopped because the workflow token budget was exhausted.")).toBeVisible();
    expect(screen.getByText("Failure category: budget_exhausted")).toBeVisible();
    expect(screen.getByText("Agent start status unknown · 1200 tokens recorded")).toBeVisible();
    expect(screen.getByText("Marginal charge: $0.025 USD.")).toBeVisible();
    expect(screen.getByText(/Choose a different mode or queue a changed selection/)).toBeVisible();
    expect(screen.queryByRole("button", { name: "Start a new job with the same selection" })).not.toBeInTheDocument();
    expect(onRefresh).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Start a new deep investigation" }));
    expect(onStartSuggestion).toHaveBeenCalledWith(selection, StagedMessageMode.DEEP);
  });
});
