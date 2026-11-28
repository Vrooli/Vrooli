import type { ExperienceSurfaceState } from "@vrooli/react-component-library/ExperienceSurface/1.0.3";
import type { Reading, SourceMetadata } from "./api";

export interface RoomSurfaceStatus {
  state: ExperienceSurfaceState;
  statusMessage?: string;
}

/**
 * Resolve the region state from the data the room actually renders. A healthy
 * integration only proves that its health endpoint answered; a signal can
 * still be unavailable, stale, or untrusted after its read completes.
 */
export function roomSurfaceStatus(readings: Reading[], sources: Record<string, SourceMetadata>): RoomSurfaceStatus {
  if (readings.length === 0) return { state: "empty" };

  const unhealthySignal = readings.some((reading) =>
    reading.coverage !== "NOW" || (reading.trust !== "VALID" && reading.trust !== "CACHED"),
  );
  const unhealthySource = Object.values(sources).some((source) =>
    Boolean(source.staleness_ts) || (source.integration_status !== undefined && source.integration_status !== "available"),
  );

  if (!unhealthySignal && !unhealthySource) return { state: "ready" };
  return {
    state: "partial",
    statusMessage: "Some readings are missing, stale, or could not be verified. Each figure shows its own source state.",
  };
}
