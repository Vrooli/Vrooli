import { EventKind, StepStatus } from '@vrooli/proto-types/browser-automation-studio/v1/base/shared_pb';
import { create, toJson } from '@bufbuild/protobuf';
import { ExecutionEventEnvelopeSchema } from '@vrooli/proto-types/browser-automation-studio/v1/execution/execution_pb';
import { describe, expect, it, vi } from 'vitest';
import {
  envelopeToExecutionEvent,
  ExecutionEventsClient,
} from '../../features/execution/ws/executionEvents';

const baseTimestamp = { seconds: BigInt(1), nanos: 0 };

describe('envelopeToExecutionEvent', () => {
  it('maps status updates into execution events', () => {
    const event = envelopeToExecutionEvent({
      executionId: 'exec-123',
      workflowId: 'wf-456',
      timestamp: baseTimestamp,
      kind: EventKind.STATUS_UPDATE,
      payload: {
        case: 'statusUpdate',
        value: { status: 2, progress: 0.25, error: '' }, // RUNNING
      },
      stepIndex: 0,
    } as any);

    expect(event?.type).toBe('execution.started');
    expect(event?.execution_id).toBe('exec-123');
    expect(event?.workflow_id).toBe('wf-456');
    expect(event?.progress).toBe(0.25);
  });

  it('maps timeline frames into step events', () => {
    const event = envelopeToExecutionEvent({
      executionId: 'exec-123',
      workflowId: 'wf-456',
      timestamp: baseTimestamp,
      kind: EventKind.TIMELINE_FRAME,
      payload: {
        case: 'timelineFrame',
        value: {
          frame: {
            stepIndex: 1,
            nodeId: 'node-1',
            stepType: 'click',
            status: StepStatus.COMPLETED,
          },
        },
      },
    } as any);

    expect(event?.type).toBe('step.completed');
    expect(event?.step_index).toBe(1);
    expect(event?.step_node_id).toBe('node-1');
    expect(event?.step_type).toBe('click');
  });
});

describe('ExecutionEventsClient', () => {
  it('dispatches parsed events to the handler', () => {
    const onEvent = vi.fn();
    const client = new ExecutionEventsClient({ onEvent });
    const listener = client.createMessageListener();

    const envelope = create(ExecutionEventEnvelopeSchema, {
      executionId: 'exec-1',
      workflowId: 'wf-1',
      timestamp: baseTimestamp,
      kind: EventKind.STATUS_UPDATE,
      payload: {
        case: 'statusUpdate',
        value: { status: 2 },
      },
    } as any);
    const payload = JSON.stringify(toJson(ExecutionEventEnvelopeSchema, envelope));

    listener({ data: payload } as unknown as MessageEvent);

    expect(onEvent).toHaveBeenCalledTimes(1);
    const [event, ctx] = onEvent.mock.calls[0];
    expect(event.type).toBe('execution.started');
    expect(ctx.fallbackTimestamp).toBeDefined();
  });

  it('dispatches legacy updates when envelopes are absent', () => {
    const onLegacy = vi.fn();
    const client = new ExecutionEventsClient({ onEvent: vi.fn(), onLegacy });
    const listener = client.createMessageListener();

    listener({ data: JSON.stringify({ type: 'completed' }) } as unknown as MessageEvent);

    expect(onLegacy).toHaveBeenCalledWith({ type: 'completed' });
  });
});
