import { useEffect, useMemo, useState } from "react";
import type { Execution } from "../store";

export type HeartbeatState =
  | "idle"
  | "awaiting"
  | "healthy"
  | "delayed"
  | "stalled";

export interface HeartbeatDescriptor {
  tone: "awaiting" | "healthy" | "delayed" | "stalled";
  iconClass: string;
  textClass: string;
  label: string;
}

export const formatSeconds = (value: number) => {
  if (Number.isNaN(value) || !Number.isFinite(value)) {
    return "0s";
  }
  if (value >= 10) {
    return `${Math.round(value)}s`;
  }
  return `${value.toFixed(1)}s`;
};

const deriveHeartbeatTimestamp = (execution: Execution) => {
  if (execution.lastHeartbeat?.timestamp) {
    return execution.lastHeartbeat.timestamp;
  }
  if (execution.completedAt) {
    return execution.completedAt;
  }
  return execution.startedAt ?? undefined;
};

export const useExecutionHeartbeat = (execution: Execution) => {
  const [heartbeatTick, setHeartbeatTick] = useState(0);
  const heartbeatTimestamp = execution.lastHeartbeat?.timestamp?.valueOf();
  const inStepSeconds =
    execution.lastHeartbeat?.elapsedMs != null
      ? Math.max(0, execution.lastHeartbeat.elapsedMs / 1000)
      : null;

  useEffect(() => {
    if (execution.status !== "running" || !heartbeatTimestamp) {
      return;
    }
    const interval = window.setInterval(() => {
      setHeartbeatTick((tick) => tick + 1);
    }, 1000);
    return () => {
      window.clearInterval(interval);
    };
  }, [execution.status, heartbeatTimestamp]);

  const derivedHeartbeatTimestamp = useMemo(
    () => deriveHeartbeatTimestamp(execution),
    [execution, execution.timeline],
  );

  const heartbeatAgeSeconds = useMemo(() => {
    if (!derivedHeartbeatTimestamp) {
      return null;
    }
    const age = (Date.now() - derivedHeartbeatTimestamp.getTime()) / 1000;
    return age < 0 ? 0 : age;
  }, [derivedHeartbeatTimestamp, heartbeatTick]);

  const heartbeatAgeLabel =
    heartbeatAgeSeconds == null
      ? null
      : heartbeatAgeSeconds < 0.75
        ? "just now"
        : `${formatSeconds(heartbeatAgeSeconds)} ago`;

  const inStepLabel =
    inStepSeconds != null ? formatSeconds(inStepSeconds) : null;

  const heartbeatState: HeartbeatState = useMemo(() => {
    const isRunning = execution.status === "running";
    const hasHeartbeat = Boolean(
      execution.lastHeartbeat || derivedHeartbeatTimestamp,
    );
    if (!hasHeartbeat) {
      return isRunning ? "awaiting" : "idle";
    }
    if (heartbeatAgeSeconds == null) {
      return "awaiting";
    }
    if (heartbeatAgeSeconds >= 15) {
      return "stalled";
    }
    if (heartbeatAgeSeconds >= 8) {
      return "delayed";
    }
    return "healthy";
  }, [execution.status, execution.lastHeartbeat, derivedHeartbeatTimestamp, heartbeatAgeSeconds]);

  const heartbeatDescriptor: HeartbeatDescriptor | null = useMemo(() => {
    const baseDescriptor = (() => {
      switch (heartbeatState) {
        case "idle":
          if (execution.status === "running") {
            return null;
          }
          return {
            tone: "awaiting" as const,
            iconClass: "text-amber-400",
            textClass: "text-amber-200/90",
            label: "No heartbeats recorded for this run",
          };
        case "awaiting":
          return {
            tone: "awaiting" as const,
            iconClass: "text-amber-400",
            textClass: "text-amber-200/90",
            label: "Awaiting first heartbeat…",
          };
        case "healthy":
          return {
            tone: "healthy" as const,
            iconClass: "text-blue-400",
            textClass: "text-blue-200",
            label: `Heartbeat ${heartbeatAgeLabel ?? "just now"}`,
          };
        case "delayed":
          return {
            tone: "delayed" as const,
            iconClass: "text-amber-400",
            textClass: "text-amber-200",
            label: `Heartbeat delayed (${formatSeconds(heartbeatAgeSeconds ?? 0)} since last update)`,
          };
        case "stalled":
          return {
            tone: "stalled" as const,
            iconClass: "text-red-400",
            textClass: "text-red-200",
            label: `Heartbeat stalled (${formatSeconds(heartbeatAgeSeconds ?? 0)} without update)`,
          };
        default:
          return null;
      }
    })();

    if (!baseDescriptor) {
      return null;
    }

    const labelWithSource = execution.lastHeartbeat
      ? baseDescriptor.label
      : `${baseDescriptor.label} (timeline activity)`;

    if (execution.status !== "running") {
      return {
        ...baseDescriptor,
        label: `${labelWithSource} • Final heartbeat snapshot`,
      };
    }

    return {
      ...baseDescriptor,
      label: labelWithSource,
    };
  }, [heartbeatState, heartbeatAgeLabel, heartbeatAgeSeconds, execution.status, execution.lastHeartbeat]);

  return {
    heartbeatDescriptor,
    heartbeatState,
    heartbeatAgeSeconds,
    heartbeatAgeLabel,
    inStepLabel,
  };
};
