import { describe, it, expect } from 'vitest';
import { create, fromJson } from '@bufbuild/protobuf';
import { ExecutionParametersSchema } from '@vrooli/proto-types/browser-automation-studio/v1/execution/execution_pb';
import { TimelineEntrySchema } from '@vrooli/proto-types/browser-automation-studio/v1/timeline/entry_pb';
import { JsonValueSchema, type JsonValue } from '@vrooli/proto-types/common/v1/types_pb';
import { captureGeometryFromParameters, mapTimelineEntryToFrame } from '@/domains/executions/store';

const asJsonValue = (value: unknown): JsonValue => {
  if (Array.isArray(value)) {
    return create(JsonValueSchema, { kind: { case: 'listValue', value: { values: value.map(asJsonValue) } } });
  }
  if (value && typeof value === 'object') {
    return create(JsonValueSchema, { kind: { case: 'objectValue', value: { fields: Object.fromEntries(
      Object.entries(value).map(([key, item]) => [key, asJsonValue(item)]),
    ) } } });
  }
  if (typeof value === 'string') return create(JsonValueSchema, { kind: { case: 'stringValue', value } });
  if (typeof value === 'number') return create(JsonValueSchema, { kind: { case: 'intValue', value: BigInt(value) } });
  return create(JsonValueSchema, { kind: { case: 'nullValue', value: 0 } });
};

describe('mapTimelineEntryToFrame', () => {
  it('normalizes proto timeline entries into replay-ready shape', () => {
    const proto = fromJson(
      TimelineEntrySchema,
      {
        id: 'entry-1',
        sequence_num: 1,
        step_index: 1,
        node_id: 'node-1',
        duration_ms: 1200,
        total_duration_ms: 1800,
        action: {
          type: 'ACTION_TYPE_NAVIGATE',
        },
        telemetry: {
          screenshot: {
            artifact_id: 'shot-1',
            url: 'https://example.test/shot.png',
            thumbnail_url: 'https://example.test/thumb.png',
            width: 800,
            height: 600,
            content_type: 'image/png',
          },
          cursor_position: { x: 5, y: 6 },
        },
        context: {
          success: true,
          retry_status: {
            current_attempt: 2,
            max_attempts: 3,
          },
        },
        aggregates: {
          status: 'STEP_STATUS_COMPLETED',
          progress: 42,
          final_url: 'https://example.test',
        },
      },
      { jsonOptions: { useProtoNames: true, ignoreUnknownFields: false } }
    );

    const mapped = mapTimelineEntryToFrame(proto);

    expect(mapped.stepIndex).toBe(1);
    expect(mapped.nodeId).toBe('node-1');
    expect(mapped.stepType).toBe('navigate');
    expect(mapped.status).toBe('completed');
    expect(mapped.durationMs).toBe(1200);
    expect(mapped.totalDurationMs).toBe(1800);
    expect(mapped.progress).toBe(42);
    expect(mapped.finalUrl).toBe('https://example.test');
    expect(mapped.screenshot?.artifactId).toBe('shot-1');
    expect(mapped.screenshot?.width).toBe(800);
    expect(mapped.cursorPosition).toMatchObject({ x: 5, y: 6 });
    expect(mapped.clickPosition).toBeNull();
    expect(mapped.cursorProvenance).toBe('missing');
    expect(mapped.retryAttempt).toBe(2);
    expect(mapped.retryMaxAttempts).toBe(3);
  });

  it('restores timestamped cursor samples from the retained step outcome artifact', () => {
    const proto = {
      id: 'entry-timed-cursor',
      stepIndex: 2,
      timestamp: { seconds: 1790992800, nanos: 125000000 },
      action: { type: 2 },
      aggregates: {
        artifacts: [{
          id: 'outcome-1',
          type: 0,
          payload: { outcome: asJsonValue({ cursor_trail: [
            { point: { x: 12, y: 34 }, recorded_at: '2026-10-03T02:00:00.125Z', elapsed_ms: 125 },
            { point: { x: 56, y: 78 }, recorded_at: '2026-10-03T02:00:00.375Z', elapsed_ms: 375 },
          ] }) },
        }],
      },
      telemetry: { cursor_trail: [{ x: 12, y: 34 }, { x: 56, y: 78 }] },
      context: { success: true },
    } as unknown as Parameters<typeof mapTimelineEntryToFrame>[0];

    const mapped = mapTimelineEntryToFrame(proto);

    expect(mapped.cursorTrailSamples).toEqual([
      { x: 12, y: 34, recordedAt: '2026-10-03T02:00:00.125Z', elapsedMs: 125 },
      { x: 56, y: 78, recordedAt: '2026-10-03T02:00:00.375Z', elapsedMs: 375 },
    ]);
    expect(mapped.observedAt).toBe('2026-10-03T02:00:00.125Z');
  });
});

describe('captureGeometryFromParameters', () => {
  it('keeps requested CSS viewport distinct from the fingerprint viewport and DPR', () => {
    const parameters = fromJson(ExecutionParametersSchema, {
      viewport_width: 1440,
      viewport_height: 900,
      browser_profile: {
        fingerprint: {
          viewport_width: 1280,
          viewport_height: 720,
          device_scale_factor: 2,
        },
      },
    }, { jsonOptions: { useProtoNames: true, ignoreUnknownFields: false } });

    expect(captureGeometryFromParameters(parameters)).toEqual({
      viewport: { width: 1440, height: 900 },
      deviceScaleFactor: 2,
    });
  });
});
