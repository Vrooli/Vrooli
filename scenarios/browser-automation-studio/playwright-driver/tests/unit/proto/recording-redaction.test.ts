import { toJson } from '@bufbuild/protobuf';
import {
  rawBrowserEventToTimelineEntry,
  timelineEntryToJson,
  TimelineEntrySchema,
  type RawBrowserEvent,
} from '../../../src/proto/recording';

function createRawInputEvent(attributes: Record<string, string>, secret: string): RawBrowserEvent {
  return {
    actionType: 'type',
    timestamp: Date.now(),
    selector: { primary: '#credential', candidates: [] },
    elementMeta: {
      tagName: 'INPUT',
      innerText: secret,
      attributes: { ...attributes, value: secret, 'data-secret': secret },
      isVisible: true,
      isEnabled: true,
    },
    url: 'https://example.test/login',
    payload: { text: secret, value: secret },
  };
}

describe('recording timeline sensitive-value redaction', () => {
  it.each([
    ['password type', { type: 'password' }],
    ['hidden type', { type: 'hidden' }],
    ['one-time-code autocomplete', { type: 'text', autocomplete: 'section-login one-time-code' }],
  ])('omits sensitive values for %s even when raw events contain them', (_label, attributes) => {
    const secret = 'BAS_SYNTHETIC_CONVERTER_SECRET_8c21';
    const entry = rawBrowserEventToTimelineEntry(createRawInputEvent(attributes, secret), {
      sessionId: 'synthetic-session',
      sequenceNum: 1,
    });
    const serialized = JSON.stringify(toJson(TimelineEntrySchema, entry));

    expect(serialized).not.toContain(secret);
    expect(entry.action?.params?.case).toBe('input');
    if (entry.action?.params?.case === 'input') {
      expect(entry.action.params.value.value).toBe('');
    }
    expect(entry.action?.metadata?.elementSnapshot?.innerText).toBe('');
    expect(entry.action?.metadata?.elementSnapshot?.attributes).not.toHaveProperty('value');
    expect(entry.action?.metadata?.elementSnapshot?.attributes).not.toHaveProperty('data-secret');
  });

  it('preserves ordinary text input values', () => {
    const value = 'ordinary workflow input';
    const entry = rawBrowserEventToTimelineEntry(createRawInputEvent({ type: 'text' }, value), {
      sessionId: 'synthetic-session',
      sequenceNum: 1,
    });

    expect(entry.action?.params?.case).toBe('input');
    if (entry.action?.params?.case === 'input') {
      expect(entry.action.params.value.value).toBe(value);
    }
  });

  it('preserves the emitting tab identity in timeline telemetry', () => {
    const entry = rawBrowserEventToTimelineEntry({
      ...createRawInputEvent({ type: 'text' }, 'value'),
      driverPageId: 'recorded-tab-two',
    }, {
      sessionId: 'synthetic-session',
      sequenceNum: 2,
    });

    expect(entry.telemetry?.driverPageId).toBe('recorded-tab-two');
    expect(timelineEntryToJson(entry).telemetry).toMatchObject({ driverPageId: 'recorded-tab-two' });
  });
});
