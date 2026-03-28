import { useCallback, useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { buildTestGenieApiUrl, type ExecuteSuiteInput, type SuiteExecutionResult } from "../lib/api";

type StreamStatus = "idle" | "streaming" | "completed" | "error";

interface StreamLogEntry {
  id: string;
  phase?: string;
  message: string;
  level: "info" | "error";
  timestamp?: string;
}

interface UseExecutionStreamOptions {
  onComplete?: (result: SuiteExecutionResult) => void;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isPhaseExecutionResultArray(value: unknown): value is SuiteExecutionResult["phases"] {
  return Array.isArray(value);
}

function isPhaseSummary(value: unknown): value is SuiteExecutionResult["phaseSummary"] {
  return (
    isRecord(value) &&
    typeof value.total === "number" &&
    typeof value.passed === "number" &&
    typeof value.failed === "number" &&
    typeof value.durationSeconds === "number" &&
    typeof value.observationCount === "number"
  );
}

function parseSSEEvent(raw: string): { event: string; data: unknown } | null {
  const lines = raw.split("\n");
  let event = "";
  let dataText = "";

  for (const line of lines) {
    if (line.startsWith("event:")) {
      event = line.slice("event:".length).trim();
    } else if (line.startsWith("data:")) {
      // Allow multiple data lines; concatenate
      const chunk = line.slice("data:".length).trim();
      dataText = dataText ? `${dataText}${chunk}` : chunk;
    }
  }

  if (!event) return null;

  let parsed: unknown = dataText;
  if (dataText) {
    try {
      parsed = JSON.parse(dataText);
    } catch {
      parsed = dataText;
    }
  }

  return { event, data: parsed };
}

export function useExecutionStream(options?: UseExecutionStreamOptions) {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<StreamStatus>("idle");
  const [logs, setLogs] = useState<StreamLogEntry[]>([]);
  const [result, setResult] = useState<SuiteExecutionResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const controllerRef = useRef<AbortController | null>(null);

  const appendLog = useCallback((entry: Omit<StreamLogEntry, "id">) => {
    setLogs((prev) => [
      ...prev,
      {
        ...entry,
        id: `${Date.now()}-${prev.length}`
      }
    ]);
  }, []);

  const reset = useCallback(() => {
    controllerRef.current?.abort();
    setStatus("idle");
    setLogs([]);
    setResult(null);
    setError(null);
  }, []);

  useEffect(() => () => controllerRef.current?.abort(), []);

  const startStream = useCallback(
    async (input: ExecuteSuiteInput) => {
      controllerRef.current?.abort();
      const controller = new AbortController();
      controllerRef.current = controller;

      setStatus("streaming");
      setLogs([]);
      setResult(null);
      setError(null);

      const url = buildTestGenieApiUrl("/executions/stream");
      const res = await fetch(url, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(input),
        signal: controller.signal
      });

      if (!res.ok || !res.body) {
        const message = !res.ok ? `Request failed with status ${res.status}` : "Streaming not supported";
        setStatus("error");
        setError(message);
        throw new Error(message);
      }

      const reader = res.body.getReader();
      const decoder = new TextDecoder();
      let buffer = "";
      let finished = false;

      const handleComplete = (data: Record<string, unknown>) => {
        const finalResult: SuiteExecutionResult = {
          scenarioName: input.scenarioName,
          success: Boolean(data.success),
          preset: typeof data.presetUsed === "string" ? data.presetUsed : input.preset,
          phases: isPhaseExecutionResultArray(data.phases) ? data.phases : [],
          phaseSummary:
            isPhaseSummary(data.phaseSummary) ? data.phaseSummary : {
              total: 0,
              passed: 0,
              failed: 0,
              durationSeconds: 0,
              observationCount: 0
            },
          startedAt: typeof data.startedAt === "string" ? data.startedAt : new Date().toISOString(),
          completedAt: typeof data.completedAt === "string" ? data.completedAt : new Date().toISOString(),
          executionId: typeof data.executionId === "string" ? data.executionId : undefined
        };
        setResult(finalResult);
        setStatus("completed");
        queryClient.invalidateQueries({ queryKey: ["executions"] });
        queryClient.invalidateQueries({ queryKey: ["health"] });
        options?.onComplete?.(finalResult);
      };

      const processEvent = (evt: string, data: Record<string, unknown>) => {
        switch (evt) {
          case "phase_start":
            appendLog({
              level: "info",
              phase: typeof data.phase === "string" ? data.phase : undefined,
              message: `Starting ${data.phase ?? "phase"} (${data.index}/${data.total})`,
              timestamp: typeof data.timestamp === "string" ? data.timestamp : undefined
            });
            break;
          case "progress":
            appendLog({
              level: "info",
              phase: typeof data.phase === "string" ? data.phase : undefined,
              message: typeof data.message === "string" ? data.message : "Progress"
            });
            break;
          case "observation":
            appendLog({
              level: "info",
              phase: typeof data.phase === "string" ? data.phase : undefined,
              message: typeof data.message === "string" ? data.message : "Observation",
              timestamp: typeof data.timestamp === "string" ? data.timestamp : undefined
            });
            break;
          case "phase_end": {
            const statusText = typeof data.status === "string" ? data.status : "done";
            const duration =
              typeof data.durationSeconds === "number" ? ` in ${data.durationSeconds}s` : "";
            const errorMsg = typeof data.error === "string" && data.error ? ` · ${data.error}` : "";
            appendLog({
              level: errorMsg ? "error" : "info",
              phase: typeof data.phase === "string" ? data.phase : undefined,
              message: `${data.phase ?? "Phase"} ${statusText}${duration}${errorMsg}`
            });
            break;
          }
          case "error": {
            const message = typeof data.message === "string" ? data.message : "Execution failed";
            appendLog({ level: "error", message });
            setError(message);
            setStatus("error");
            finished = true;
            controller.abort();
            break;
          }
          case "complete":
            handleComplete(data);
            finished = true;
            controller.abort();
            break;
          default:
            break;
        }
      };

      try {
        while (true) {
          const { value, done } = await reader.read();
          if (done || finished) {
            break;
          }

          buffer += decoder.decode(value, { stream: true });

          let delimiterIndex: number;
          while ((delimiterIndex = buffer.indexOf("\n\n")) !== -1) {
            const rawEvent = buffer.slice(0, delimiterIndex);
            buffer = buffer.slice(delimiterIndex + 2);
            const parsed = parseSSEEvent(rawEvent);
            if (parsed && isRecord(parsed.data)) {
              processEvent(parsed.event, parsed.data);
            }
          }
        }
        if (!finished) {
          const message = "Stream ended before completion event";
          setError(message);
          setStatus("error");
          appendLog({ level: "error", message });
        }
      } catch (err) {
        if (err instanceof Error && err.name === "AbortError") {
          return;
        }
        const message = err instanceof Error ? err.message : "Stream terminated unexpectedly";
        setError(message);
        setStatus("error");
        appendLog({ level: "error", message });
        throw err;
      }
    },
    [appendLog, options, queryClient]
  );

  return {
    startStream,
    reset,
    status,
    logs,
    result,
    error
  };
}
