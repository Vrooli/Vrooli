import { useEffect, useRef, useState } from "react";
import { type QueryClient } from "@tanstack/react-query";

import {
  getExperiment,
  streamExperimentEvents,
  type ExperimentEventRow,
  type ExperimentRow,
} from "../../services/experiment";
import { isTerminal } from "./ExperimentLabFormat";

interface UseExperimentStreamArgs {
  selectedExperimentId: string;
  selectedExperimentStatus: ExperimentRow["status"];
  selectedId: string;
  queryClient: QueryClient;
  loadReport: (id: string) => void;
  messages: {
    complete: string;
    polling: string;
    streamClosed: string;
  };
  setActiveExperiment: (experiment: ExperimentRow) => void;
}

export function useExperimentStream({
  selectedExperimentId,
  selectedExperimentStatus,
  selectedId,
  queryClient,
  loadReport,
  messages,
  setActiveExperiment,
}: UseExperimentStreamArgs) {
  const [liveEvent, setLiveEvent] = useState<ExperimentEventRow | null>(null);
  const [streamError, setStreamError] = useState("");
  const qcRef = useRef(queryClient);
  const loadReportRef = useRef(loadReport);
  const messagesRef = useRef(messages);
  qcRef.current = queryClient;
  loadReportRef.current = loadReport;
  messagesRef.current = messages;

  useEffect(() => {
    if (!selectedExperimentId || isTerminal(selectedExperimentStatus)) return;
    const controller = new AbortController();
    let closed = false;
    let fallbackTimer: number | null = null;

    const handleTerminal = (id: string) => {
      void qcRef.current.invalidateQueries({ queryKey: ["experiments", "list"] });
      loadReportRef.current(id);
    };

    const startFallbackPolling = () => {
      if (closed || fallbackTimer) return;
      fallbackTimer = window.setInterval(() => {
        void getExperiment(selectedExperimentId)
          .then(({ experiment }) => {
            if (!experiment || closed) return;
            setActiveExperiment(experiment);
            setLiveEvent((current) => ({
              experimentId: experiment.id,
              status: experiment.status,
              progress: isTerminal(experiment.status) ? 100 : current?.experimentId === experiment.id ? current.progress : 0,
              message: isTerminal(experiment.status) ? messagesRef.current.complete : messagesRef.current.polling,
              at: experiment.finishedAt || experiment.startedAt || experiment.createdAt,
            }));
            if (isTerminal(experiment.status)) {
              handleTerminal(experiment.id);
              if (fallbackTimer) window.clearInterval(fallbackTimer);
              fallbackTimer = null;
            }
          })
          .catch((error: unknown) => {
            if (!closed) setStreamError(error instanceof Error ? error.message : String(error));
          });
      }, 2500);
    };

    setStreamError("");
    let terminalSeen = false;
    void streamExperimentEvents(
      selectedExperimentId,
      (event) => {
        if (closed) return;
        setLiveEvent(event);
        if (isTerminal(event.status)) {
          terminalSeen = true;
          handleTerminal(event.experimentId);
          controller.abort();
        }
      },
      controller.signal,
    )
      .then(() => {
        if (closed || controller.signal.aborted || terminalSeen) return;
        setStreamError(messagesRef.current.streamClosed);
        startFallbackPolling();
      })
      .catch((error: unknown) => {
        if (closed || controller.signal.aborted) return;
        setStreamError(error instanceof Error ? error.message : String(error));
        startFallbackPolling();
      });

    return () => {
      closed = true;
      controller.abort();
      if (fallbackTimer) window.clearInterval(fallbackTimer);
    };
  }, [selectedExperimentId, selectedExperimentStatus, setActiveExperiment]);

  return {
    liveEvent: liveEvent?.experimentId === selectedId ? liveEvent : null,
    setLiveEvent,
    streamError,
    setStreamError,
  };
}
