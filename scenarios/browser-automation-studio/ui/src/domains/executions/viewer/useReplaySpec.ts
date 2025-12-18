import { useCallback, useEffect, useRef, useState } from "react";
import type { Execution } from "@stores/executionStore";
import type { ReplayMovieSpec } from "@/types/export";
import {
  describePreviewStatusMessage,
  normalizePreviewStatus,
} from "../utils/exportHelpers";
import {
  fetchExecutionExportPreview,
  type ExportPreviewMetrics,
} from "./exportPreview";
import { logger } from "@utils/logger";

const MOVIE_SPEC_POLL_INTERVAL_MS = 4000;

interface UseReplaySpecOptions {
  executionId: string;
  executionStatus: Execution["status"];
  timelineLength?: number;
}

export const useReplaySpec = ({
  executionId,
  executionStatus,
  timelineLength,
}: UseReplaySpecOptions) => {
  const [movieSpec, setMovieSpec] = useState<ReplayMovieSpec | null>(null);
  const [movieSpecError, setMovieSpecError] = useState<string | null>(null);
  const [isMovieSpecLoading, setIsMovieSpecLoading] = useState(false);
  const [previewMetrics, setPreviewMetrics] = useState<ExportPreviewMetrics>({
    capturedFrames: 0,
    assetCount: 0,
    totalDurationMs: 0,
  });
  const [activeSpecId, setActiveSpecId] = useState<string | null>(null);
  const movieSpecRetryTimeoutRef = useRef<number | null>(null);
  const movieSpecAbortControllerRef = useRef<AbortController | null>(null);

  const clearMovieSpecRetryTimeout = useCallback(() => {
    if (movieSpecRetryTimeoutRef.current != null) {
      window.clearTimeout(movieSpecRetryTimeoutRef.current);
      movieSpecRetryTimeoutRef.current = null;
    }
  }, []);

  useEffect(() => {
    return () => {
      clearMovieSpecRetryTimeout();
      if (movieSpecAbortControllerRef.current) {
        movieSpecAbortControllerRef.current.abort();
        movieSpecAbortControllerRef.current = null;
      }
    };
  }, [clearMovieSpecRetryTimeout]);

  useEffect(() => {
    let isCancelled = false;

    if (movieSpecAbortControllerRef.current) {
      movieSpecAbortControllerRef.current.abort();
      movieSpecAbortControllerRef.current = null;
    }
    clearMovieSpecRetryTimeout();

    async function fetchMovieSpec() {
      if (isCancelled) {
        return;
      }

      const abortController = new AbortController();
      movieSpecAbortControllerRef.current = abortController;

      let shouldRetry = false;
      try {
        setIsMovieSpecLoading(true);
        const {
          preview,
          status,
          metrics,
          movieSpec,
        } = await fetchExecutionExportPreview(executionId, {
          signal: abortController.signal,
        });
        const specId =
          (preview.specId && preview.specId.trim()) ||
          preview.executionId ||
          executionId;

        setPreviewMetrics(metrics);
        setActiveSpecId(specId);

        const message = describePreviewStatusMessage(
          status,
          preview.message,
          metrics,
        );
        if (status !== "ready") {
          setMovieSpec(null);
          setMovieSpecError(message);
          shouldRetry = normalizePreviewStatus(status) === "pending";
          return;
        }
        if (!movieSpec) {
          setMovieSpec(null);
          setMovieSpecError(
            "Replay export unavailable – missing export package",
          );
          return;
        }
        clearMovieSpecRetryTimeout();
        setMovieSpec(movieSpec);
        setMovieSpecError(null);
      } catch (error) {
        if (isCancelled) {
          return;
        }
        if (error instanceof DOMException && error.name === "AbortError") {
          return;
        }
        const message =
          error instanceof Error ? error.message : "Failed to load replay spec";
        setMovieSpecError(message);
        setMovieSpec(null);
        logger.error(
          "Failed to load replay movie spec",
          { component: "ExecutionViewer", executionId },
          error,
        );
        shouldRetry = executionStatus === "running";
      } finally {
        if (!isCancelled) {
          setIsMovieSpecLoading(false);
        }
        if (!isCancelled) {
          movieSpecAbortControllerRef.current = null;
        }
        if (!isCancelled && shouldRetry) {
          clearMovieSpecRetryTimeout();
          movieSpecRetryTimeoutRef.current = window.setTimeout(() => {
            void fetchMovieSpec();
          }, MOVIE_SPEC_POLL_INTERVAL_MS);
        }
      }
    }

    setMovieSpecError(null);
    setMovieSpec(null);
    setIsMovieSpecLoading(true);
    void fetchMovieSpec();

    return () => {
      isCancelled = true;
      clearMovieSpecRetryTimeout();
      if (movieSpecAbortControllerRef.current) {
        movieSpecAbortControllerRef.current.abort();
        movieSpecAbortControllerRef.current = null;
      }
    };
  }, [
    executionId,
    executionStatus,
    timelineLength,
    clearMovieSpecRetryTimeout,
  ]);

  return {
    movieSpec,
    movieSpecError,
    isMovieSpecLoading,
    previewMetrics,
    setPreviewMetrics,
    activeSpecId,
    setActiveSpecId,
    setMovieSpecError,
  };
};
