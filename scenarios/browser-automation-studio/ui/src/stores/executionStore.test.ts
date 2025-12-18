import { describe, it, expect } from 'vitest';
import { fromJson } from '@bufbuild/protobuf';
import { TimelineEntrySchema } from '@vrooli/proto-types/browser-automation-studio/v1/timeline/entry_pb';
import { mapTimelineEntryToFrame } from '@/domains/executions';

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
          dom_snapshot_preview: '<html>preview</html>',
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
    // cursorPosition maps to clickPosition in the new schema
    expect(mapped.clickPosition).toMatchObject({ x: 5, y: 6 });
    expect(mapped.retryAttempt).toBe(2);
    expect(mapped.retryMaxAttempts).toBe(3);
    expect(mapped.domSnapshotPreview).toBe('<html>preview</html>');
  });
});
