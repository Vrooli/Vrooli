import { create, toJson } from '@bufbuild/protobuf';
import {
  BoundingBoxSchema,
} from '@vrooli/proto-types/browser-automation-studio/v1/base/geometry_pb';
import {
  ElementMetaSchema,
} from '@vrooli/proto-types/browser-automation-studio/v1/domain/selectors_pb';
import {
  StepOutcomeSchema,
} from '@vrooli/proto-types/browser-automation-studio/v1/execution/driver_pb';
import {
  buildStepOutcome,
  toDriverOutcome,
  type HandlerResult,
  type BuildOutcomeParams,
} from '../../../src/outcome/outcome-builder';
import { createTestInstruction } from '../../helpers';

describe('Outcome Builder', () => {
  it('builds a StepOutcome with telemetry, focus, and failure details', () => {
    const instruction = createTestInstruction({
      index: -1,
      nodeId: 'node-1',
      type: 'click',
    });

    const elementMeta = create(ElementMetaSchema, {
      tagName: 'button',
      isVisible: true,
      isEnabled: true,
    });

    const boundingBox = create(BoundingBoxSchema, {
      x: 10,
      y: 20,
      width: 100,
      height: 40,
    });

    const result: HandlerResult = {
      success: false,
      error: {
        message: 'boom',
        code: 'INTERNAL_ERROR',
        kind: 'engine',
        retryable: false,
      },
      extracted_data: { key: 'value' },
      focus: {
        selector: '#target',
        bounding_box: { x: 1, y: 2, width: 3, height: 4 },
      },
      elementContext: {
        selector: '#target',
        confidence: 0.8,
        matchCount: 2,
        elementMeta,
        boundingBox,
      },
    };

    const params: BuildOutcomeParams = {
      instruction,
      result,
      startedAt: new Date('2024-01-01T00:00:00.000Z'),
      completedAt: new Date('2024-01-01T00:00:01.000Z'),
      finalUrl: 'https://example.com',
      screenshot: {
        base64: Buffer.from('image-bytes').toString('base64'),
        width: 800,
        height: 600,
        capture_time: '2024-01-01T00:00:00.500Z',
      },
      domSnapshot: {
        html: '<html></html>',
        preview: '<html>',
        collected_at: '2024-01-01T00:00:00.600Z',
      },
      consoleLogs: [
        { type: 'log', text: 'hello', timestamp: '2024-01-01T00:00:00.700Z' },
      ],
      networkEvents: [
        { type: 'request', url: 'https://example.com', timestamp: '2024-01-01T00:00:00.800Z' },
      ],
    };

    const outcome = buildStepOutcome(params);
    const json = toJson(StepOutcomeSchema, outcome);

    expect(outcome.stepIndex).toBe(0);
    expect(outcome.success).toBe(false);
    expect(outcome.failure?.message).toBe('boom');
    expect(outcome.screenshot).toBeDefined();
    expect(outcome.domSnapshot).toBeDefined();
    expect(outcome.consoleLogs.length).toBe(1);
    expect(outcome.networkEvents.length).toBe(1);
    expect(outcome.focusedElement?.selector).toBe('#target');
    expect(outcome.usedSelector).toBe('#target');
    expect(outcome.selectorConfidence).toBe(0.8);
    expect(outcome.selectorMatchCount).toBe(2);
    expect(outcome.elementSnapshot).toBeDefined();
    expect(outcome.elementBoundingBox).toBeDefined();
    expect(json.extractedData).toBeDefined();
  });

  it('flattens driver outcome fields for backward compatibility', () => {
    const instruction = createTestInstruction({
      index: 0,
      nodeId: 'node-2',
      type: 'screenshot',
    });

    const result: HandlerResult = {
      success: true,
    };

    const outcome = buildStepOutcome({
      instruction,
      result,
      startedAt: new Date('2024-01-01T00:00:00.000Z'),
      completedAt: new Date('2024-01-01T00:00:00.500Z'),
      finalUrl: 'https://example.com',
    });

    const driverOutcome = toDriverOutcome(
      outcome,
      {
        base64: 'base64-data',
        media_type: 'image/png',
        width: 100,
        height: 200,
      },
      {
        html: '<html>',
        preview: '<html>',
      }
    );

    expect(driverOutcome.screenshot_base64).toBe('base64-data');
    expect(driverOutcome.screenshot_media_type).toBe('image/png');
    expect(driverOutcome.screenshot_width).toBe(100);
    expect(driverOutcome.screenshot_height).toBe(200);
    expect(driverOutcome.dom_html).toBe('<html>');
    expect(driverOutcome.dom_preview).toBe('<html>');
  });
});
