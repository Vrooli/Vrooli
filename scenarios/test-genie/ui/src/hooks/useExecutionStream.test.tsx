import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactElement } from "react";
import { renderWithProviders } from "../test-utils";
import { useExecutionStream } from "./useExecutionStream";
import type { ExecuteSuiteInput, SuiteExecutionResult } from "../lib/api";

const fetchMock = vi.fn<typeof fetch>();
const input: ExecuteSuiteInput = {
  scenarioName: "demo",
  preset: "smoke",
  failFast: false,
};

function renderWithQueryClient(ui: ReactElement) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  const result = renderWithProviders(ui, {
    withoutQueryClient: true,
    withoutRouter: true,
    extraProviders: (children) => <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>,
  });
  return { ...result, queryClient };
}

function responseWithChunks(chunks: string[]): Response {
  const encoder = new TextEncoder();
  const body = new ReadableStream<Uint8Array>({
    start(controller) {
      for (const chunk of chunks) controller.enqueue(encoder.encode(chunk));
      controller.close();
    },
  });
  return new Response(body, { status: 200, headers: { "Content-Type": "text/event-stream" } });
}

function chunkedSSE(events: Array<{ event: string; data: Record<string, unknown> }>): Response {
  const wire = encodeSSE(events);
  // Deliberately split event names, JSON, and frame delimiters across reads.
  const chunks: string[] = [];
  for (let offset = 0; offset < wire.length; offset += 11) {
    chunks.push(wire.slice(offset, offset + 11));
  }
  return responseWithChunks(chunks);
}

function encodeSSE(events: Array<{ event: string; data: Record<string, unknown> }>): string {
  return events
    .map(({ event, data }) => `event: ${event}\ndata: ${JSON.stringify(data)}\n\n`)
    .join("");
}

function StreamProbe({
  onComplete,
  onStartError,
}: {
  onComplete?: (result: SuiteExecutionResult) => void;
  onStartError?: (error: unknown) => void;
}) {
  const stream = useExecutionStream({ onComplete });
  return (
    <section>
      <button type="button" onClick={() => void stream.startStream(input).catch((error) => onStartError?.(error))}>Start</button>
      <button type="button" onClick={stream.reset}>Reset</button>
      <output data-testid="status">{stream.status}</output>
      <output data-testid="error">{stream.error ?? ""}</output>
      <ul data-testid="logs">
        {stream.logs.map((entry) => <li key={entry.id}>{entry.phase ? `[${entry.phase}] ` : ""}{entry.message}</li>)}
      </ul>
      <output data-testid="result">{stream.result ? JSON.stringify(stream.result) : ""}</output>
    </section>
  );
}

const completeEvent = {
  success: true,
  presetUsed: "smoke",
  phases: [{ name: "unit", status: "passed", durationSeconds: 2, logPath: "unit.log" }],
  phaseSummary: { total: 1, passed: 1, failed: 0, durationSeconds: 2, observationCount: 0 },
  startedAt: "2026-10-20T10:00:00Z",
  completedAt: "2026-10-20T10:00:02Z",
  executionId: "execution-1",
};

describe("useExecutionStream", () => {
  beforeEach(() => {
    fetchMock.mockReset();
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
  });

  it("posts the selected input, consumes split SSE frames, and invalidates execution summaries on completion", async () => {
    fetchMock.mockResolvedValue(chunkedSSE([
      { event: "phase_start", data: { phase: "unit", index: 1, total: 1, timestamp: "10:00:00" } },
      { event: "progress", data: { phase: "unit", message: "checking files" } },
      { event: "phase_end", data: { phase: "unit", status: "passed", durationSeconds: 2 } },
      { event: "complete", data: completeEvent },
    ]));
    const onComplete = vi.fn();
    const { queryClient } = renderWithQueryClient(<StreamProbe onComplete={onComplete} />);
    queryClient.setQueryData(["executions"], [{ executionId: "old" }]);
    queryClient.setQueryData(["health"], { runCount: 1 });

    fireEvent.click(screen.getByRole("button", { name: "Start" }));

    await waitFor(() => expect(screen.getByTestId("status")).toHaveTextContent("completed"));
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/executions/stream"),
      expect.objectContaining({
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(input),
      }),
    );
    expect(screen.getByTestId("logs")).toHaveTextContent("Starting unit (1/1)");
    expect(screen.getByTestId("logs")).toHaveTextContent("[unit] checking files");
    expect(screen.getByTestId("logs")).toHaveTextContent("[unit] unit passed in 2s");
    expect(JSON.parse(screen.getByTestId("result").textContent ?? "{}")).toMatchObject({
      scenarioName: "demo",
      executionId: "execution-1",
      success: true,
      phaseSummary: { total: 1, passed: 1 },
    });
    expect(onComplete).toHaveBeenCalledWith(expect.objectContaining({ executionId: "execution-1", success: true }));
    await waitFor(() => {
      expect(queryClient.getQueryState(["executions"])?.isInvalidated).toBe(true);
      expect(queryClient.getQueryState(["health"])?.isInvalidated).toBe(true);
    });
  });

  it("renders a server error event and does not report completion", async () => {
    fetchMock.mockResolvedValue(chunkedSSE([
      { event: "error", data: { message: "plan is no longer valid" } },
    ]));
    const onComplete = vi.fn();
    renderWithQueryClient(<StreamProbe onComplete={onComplete} />);

    fireEvent.click(screen.getByRole("button", { name: "Start" }));

    await waitFor(() => expect(screen.getByTestId("status")).toHaveTextContent("error"));
    expect(screen.getByTestId("error")).toHaveTextContent("plan is no longer valid");
    expect(screen.getByTestId("logs")).toHaveTextContent("plan is no longer valid");
    expect(onComplete).not.toHaveBeenCalled();
  });

  it("keeps the completed result when the completion callback throws and rejects that callback error", async () => {
    let requestSignal: AbortSignal | undefined;
    fetchMock.mockImplementation((_request, init) => {
      requestSignal = init?.signal ?? undefined;
      return Promise.resolve(chunkedSSE([{ event: "complete", data: completeEvent }]));
    });
    const callbackError = new Error("completion consumer failed");
    const onComplete = vi.fn(() => { throw callbackError; });
    const onStartError = vi.fn();
    renderWithQueryClient(<StreamProbe onComplete={onComplete} onStartError={onStartError} />);

    fireEvent.click(screen.getByRole("button", { name: "Start" }));

    await waitFor(() => expect(screen.getByTestId("status")).toHaveTextContent("completed"));
    await waitFor(() => expect(onStartError).toHaveBeenCalledWith(callbackError));
    expect(requestSignal?.aborted).toBe(true);
    expect(screen.getByTestId("result")).toHaveTextContent("execution-1");
    expect(screen.getByTestId("error")).toBeEmptyDOMElement();
    expect(screen.getByTestId("logs")).not.toHaveTextContent(callbackError.message);
  });

  it("moves to an error state when the stream request itself rejects", async () => {
    fetchMock.mockRejectedValue(new Error("network unavailable"));
    renderWithQueryClient(<StreamProbe />);

    fireEvent.click(screen.getByRole("button", { name: "Start" }));

    await waitFor(() => expect(screen.getByTestId("status")).toHaveTextContent("error"));
    expect(screen.getByTestId("error")).toHaveTextContent("network unavailable");
    expect(screen.getByTestId("logs")).toHaveTextContent("network unavailable");
  });

  it.each([
    {
      name: "error before complete",
      events: [
        { event: "error", data: { message: "execution failed" } },
        { event: "complete", data: completeEvent },
      ],
      status: "error",
      completeCount: 0,
    },
    {
      name: "complete before error",
      events: [
        { event: "complete", data: completeEvent },
        { event: "error", data: { message: "late error" } },
      ],
      status: "completed",
      completeCount: 1,
    },
  ])("keeps the first terminal event when a frame follows it in the same chunk ($name)", async ({ events, status, completeCount }) => {
    fetchMock.mockResolvedValue(responseWithChunks([encodeSSE(events)]));
    const onComplete = vi.fn();
    renderWithQueryClient(<StreamProbe onComplete={onComplete} />);

    fireEvent.click(screen.getByRole("button", { name: "Start" }));

    await waitFor(() => expect(screen.getByTestId("status")).toHaveTextContent(status));
    expect(onComplete).toHaveBeenCalledTimes(completeCount);
    expect(screen.getByTestId("error")).toHaveTextContent(status === "error" ? "execution failed" : "");
  });

  it("reports premature EOF instead of treating an incomplete stream as a completed run", async () => {
    fetchMock.mockResolvedValue(chunkedSSE([
      { event: "progress", data: { message: "still running" } },
    ]));
    const onComplete = vi.fn();
    renderWithQueryClient(<StreamProbe onComplete={onComplete} />);

    fireEvent.click(screen.getByRole("button", { name: "Start" }));

    await waitFor(() => expect(screen.getByTestId("error")).toHaveTextContent("Stream ended before completion event"));
    expect(screen.getByTestId("status")).toHaveTextContent("error");
    expect(onComplete).not.toHaveBeenCalled();
  });

  it("aborts the active request when reset is selected", async () => {
    let requestSignal: AbortSignal | undefined;
    fetchMock.mockImplementation((_input, init) => {
      requestSignal = init?.signal ?? undefined;
      let controller: ReadableStreamDefaultController<Uint8Array>;
      const body = new ReadableStream<Uint8Array>({
        start(streamController) {
          controller = streamController;
          requestSignal?.addEventListener("abort", () => {
            controller.error(Object.assign(new Error("aborted"), { name: "AbortError" }));
          }, { once: true });
        },
      });
      return Promise.resolve(new Response(body, { status: 200 }));
    });
    renderWithQueryClient(<StreamProbe />);

    fireEvent.click(screen.getByRole("button", { name: "Start" }));
    await waitFor(() => expect(screen.getByTestId("status")).toHaveTextContent("streaming"));
    fireEvent.click(screen.getByRole("button", { name: "Reset" }));

    expect(requestSignal?.aborted).toBe(true);
    expect(screen.getByTestId("status")).toHaveTextContent("idle");
    expect(screen.getByTestId("error")).toBeEmptyDOMElement();
  });

  it("ignores a late response from a replaced, aborted request", async () => {
    let resolveFirst!: (response: Response) => void;
    let firstSignal: AbortSignal | undefined;
    fetchMock
      .mockImplementationOnce((_input, init) => {
        firstSignal = init?.signal ?? undefined;
        return new Promise<Response>((resolve) => { resolveFirst = resolve; });
      })
      .mockResolvedValueOnce(chunkedSSE([
        { event: "complete", data: { ...completeEvent, executionId: "replacement-execution" } },
      ]));
    renderWithQueryClient(<StreamProbe />);

    fireEvent.click(screen.getByRole("button", { name: "Start" }));
    await waitFor(() => expect(screen.getByTestId("status")).toHaveTextContent("streaming"));
    fireEvent.click(screen.getByRole("button", { name: "Start" }));
    await waitFor(() => expect(screen.getByTestId("status")).toHaveTextContent("completed"));

    expect(firstSignal?.aborted).toBe(true);
    resolveFirst(chunkedSSE([
      { event: "error", data: { message: "late response from old request" } },
    ]));
    await waitFor(() => expect(screen.getByTestId("result")).toHaveTextContent("replacement-execution"));
    expect(screen.getByTestId("status")).toHaveTextContent("completed");
    expect(screen.getByTestId("error")).toBeEmptyDOMElement();
  });
});
