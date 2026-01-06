/**
 * Artifact Extraction Utilities
 *
 * Pure functions for extracting typed artifacts from timeline frames.
 * Used by RecordingSession and other components that need to parse
 * timeline artifacts.
 */

import type { TimelineFrame } from '@/domains/executions';
import type { ConsoleLogEntry, NetworkEventEntry, DomSnapshotEntry } from '../sidebar/ArtifactsTab';

/**
 * Extract console logs from timeline artifacts.
 * Console logs are captured as artifacts with type 'console_log'.
 */
export function extractConsoleLogs(timeline: TimelineFrame[]): ConsoleLogEntry[] {
  const consoleLogs: ConsoleLogEntry[] = [];
  for (const frame of timeline) {
    if (!frame.artifacts) continue;
    for (const artifact of frame.artifacts) {
      if (artifact.type === 'console_log' && artifact.payload) {
        const payload = artifact.payload as {
          level?: string;
          text?: string;
          timestamp?: string;
          stack?: string;
          location?: string;
        };
        consoleLogs.push({
          id: artifact.id,
          level: (payload.level as ConsoleLogEntry['level']) ?? 'log',
          text: payload.text ?? '',
          timestamp: payload.timestamp ? new Date(payload.timestamp) : new Date(),
          stack: payload.stack,
          location: payload.location,
        });
      }
    }
  }
  return consoleLogs;
}

/**
 * Extract network events from timeline artifacts.
 * Network events are captured as artifacts with type 'network_event'.
 */
export function extractNetworkEvents(timeline: TimelineFrame[]): NetworkEventEntry[] {
  const networkEvents: NetworkEventEntry[] = [];
  for (const frame of timeline) {
    if (!frame.artifacts) continue;
    for (const artifact of frame.artifacts) {
      if (artifact.type === 'network_event' && artifact.payload) {
        const payload = artifact.payload as {
          type?: string;
          url?: string;
          method?: string;
          resourceType?: string;
          status?: number;
          ok?: boolean;
          failure?: string;
          timestamp?: string;
          durationMs?: number;
        };
        networkEvents.push({
          id: artifact.id,
          type: (payload.type as NetworkEventEntry['type']) ?? 'request',
          url: payload.url ?? '',
          method: payload.method,
          resourceType: payload.resourceType,
          status: payload.status,
          ok: payload.ok,
          failure: payload.failure,
          timestamp: payload.timestamp ? new Date(payload.timestamp) : new Date(),
          durationMs: payload.durationMs,
        });
      }
    }
  }
  return networkEvents;
}

/**
 * Extract DOM snapshots from timeline artifacts.
 * DOM snapshots are captured as artifacts with type 'dom_snapshot'.
 */
export function extractDomSnapshots(timeline: TimelineFrame[]): DomSnapshotEntry[] {
  const snapshots: DomSnapshotEntry[] = [];
  for (const frame of timeline) {
    if (!frame.artifacts) continue;
    for (const artifact of frame.artifacts) {
      if (artifact.type === 'dom_snapshot') {
        snapshots.push({
          id: artifact.id,
          stepIndex: artifact.step_index ?? frame.stepIndex ?? 0,
          stepName: artifact.label ?? frame.stepType ?? 'Step',
          timestamp: new Date(),
          storageUrl: artifact.storage_url,
          sizeBytes: artifact.size_bytes,
        });
      }
    }
  }
  return snapshots;
}
