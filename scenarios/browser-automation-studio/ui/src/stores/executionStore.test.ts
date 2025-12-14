import { describe, it, expect } from 'vitest';
import { fromJson } from '@bufbuild/protobuf';
import { TimelineFrameSchema } from '@vrooli/proto-types/browser-automation-studio/v1/timeline/container_pb';
import { mapTimelineFrameFromProto } from './executionStore';

describe('mapTimelineFrameFromProto', () => {
  it('normalizes proto timeline frames into replay-ready shape', () => {
    const proto = fromJson(
      TimelineFrameSchema,
      {
        step_index: 1,
        node_id: 'node-1',
        step_type: 'STEP_TYPE_NAVIGATE',
        status: 'STEP_STATUS_COMPLETED',
        success: true,
        duration_ms: 1200,
        total_duration_ms: 1800,
        progress: 42,
        final_url: 'https://example.test',
        highlight_regions: [
          {
            selector: '#cta',
            bounding_box: { x: 1, y: 2, width: 100, height: 80 },
            padding: 4,
          },
        ],
        cursor_trail: [{ x: 5, y: 6 }],
        screenshot: {
          artifact_id: 'shot-1',
          url: 'https://example.test/shot.png',
          thumbnail_url: 'https://example.test/thumb.png',
          width: 800,
          height: 600,
          content_type: 'image/png',
        },
        retry_attempt: 2,
        retry_max_attempts: 3,
        dom_snapshot_preview: '<html>preview</html>',
      },
      { jsonOptions: { useProtoNames: true, ignoreUnknownFields: false } }
    );

    const mapped = mapTimelineFrameFromProto(proto);

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
    expect(mapped.highlightRegions?.[0]?.boundingBox).toMatchObject({ x: 1, y: 2, width: 100, height: 80 });
    expect(mapped.cursorTrail?.[0]).toMatchObject({ x: 5, y: 6 });
    expect(mapped.retryAttempt).toBe(2);
    expect(mapped.retryMaxAttempts).toBe(3);
    expect(mapped.domSnapshotPreview).toBe('<html>preview</html>');
  });
});
