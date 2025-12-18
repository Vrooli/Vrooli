import { create, toJson } from '@bufbuild/protobuf';
import {
  TimelineMessageType,
  TimelineStreamMessageSchema,
} from '@vrooli/proto-types/browser-automation-studio/v1/timeline/entry_pb';
import {
  ExecutionStatus,
  StepStatus,
} from '@vrooli/proto-types/browser-automation-studio/v1/base/shared_pb';
import { ActionType } from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
import { describe, expect, it, vi } from 'vitest';
import {
  envelopeToExecutionEvent,
  ExecutionEventsClient,
} from '../../domains/executions/live/executionEvents';

const baseTimestamp = { seconds: BigInt(1), nanos: 0 };

describe('envelopeToExecutionEvent', () => {
  it('maps status updates into execution events', () => {
    const event = envelopeToExecutionEvent({
      payload: {
        case: 'status',
        value: { id: 'exec-123', status: ExecutionStatus.RUNNING, progress: 25, error: '' },
      },
      type: TimelineMessageType.TIMELINE_MESSAGE_TYPE_STATUS,
    } as any);

    expect(event?.type).toBe('execution.started');
    expect(event?.execution_id).toBe('exec-123');
    expect(event?.progress).toBe(25);
  });

  it('maps timeline frames into step events', () => {
    const event = envelopeToExecutionEvent({
      payload: {
        case: 'entry',
        value: {
          stepIndex: 1,
          nodeId: 'node-1',
          action: { type: ActionType.CLICK },
          aggregates: { status: StepStatus.COMPLETED, progress: 1 },
        },
      },
      type: TimelineMessageType.TIMELINE_MESSAGE_TYPE_ENTRY,
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

    const envelope = create(TimelineStreamMessageSchema, {
      type: TimelineMessageType.TIMELINE_MESSAGE_TYPE_STATUS,
      payload: {
        case: 'status',
        value: { id: 'exec-1', status: ExecutionStatus.RUNNING, progress: 25 },
      },
    } as any);
    const payload = JSON.stringify(toJson(TimelineStreamMessageSchema, envelope));

    listener({ data: payload } as unknown as MessageEvent);

    expect(onEvent).toHaveBeenCalledTimes(1);
    const [event, ctx] = onEvent.mock.calls[0];
    expect(event.type).toBe('execution.started');
    expect(ctx.fallbackTimestamp).toBeUndefined();
    expect(ctx.fallbackProgress).toBeDefined();
  });

  it('dispatches legacy updates when envelopes are absent', () => {
    const onLegacy = vi.fn();
    const client = new ExecutionEventsClient({ onEvent: vi.fn(), onLegacy });
    const listener = client.createMessageListener();

    listener({ data: JSON.stringify({ type: 'completed' }) } as unknown as MessageEvent);

    expect(onLegacy).toHaveBeenCalledWith({ type: 'completed' });
  });
});
