import {
  MAX_HISTORY_SCREENSHOT_BYTES,
  trimConversationHistory,
} from '../../../../src/ai/vision-agent/conversation-history';
import type { ConversationMessageInterface } from '../../../../src/ai/vision-agent/types';

describe('trimConversationHistory', () => {
  it('preserves the system prompt, goal, and latest observation while bounding retained images', () => {
    const history: ConversationMessageInterface[] = [
      { role: 'system', content: 'system prompt' },
      { role: 'user', content: 'Goal: finish task' },
      { role: 'user', content: 'Current state (step 1): old', screenshot: Buffer.alloc(1_500_000) },
      {
        role: 'user',
        content: 'Current state (step 2): recent',
        screenshot: Buffer.alloc(1_500_000),
      },
      { role: 'user', content: 'Current state (step 3): now', screenshot: Buffer.alloc(900_000) },
    ];

    trimConversationHistory(history, 20);

    expect(history.map((message) => message.content)).toEqual([
      'system prompt',
      'Goal: finish task',
      'Current state (step 1): old',
      'Current state (step 2): recent',
      'Current state (step 3): now',
    ]);
    expect(history[2]?.screenshot).toBeUndefined();
    expect(history[3]?.screenshot?.byteLength).toBe(1_500_000);
    expect(history[4]?.screenshot).toBeUndefined();
    expect(
      history.reduce((bytes, message) => bytes + (message.screenshot?.byteLength ?? 0), 0)
    ).toBeLessThanOrEqual(MAX_HISTORY_SCREENSHOT_BYTES);
  });

  it('keeps the newest messages when the count bound is exceeded', () => {
    const history: ConversationMessageInterface[] = [
      { role: 'system', content: 'system prompt' },
      { role: 'user', content: 'Goal: finish task' },
      { role: 'user', content: 'Current state (step 1): old' },
      { role: 'user', content: 'Current state (step 2): recent' },
      { role: 'user', content: 'Current state (step 3): now', screenshot: Buffer.from('current') },
    ];

    trimConversationHistory(history, 4);

    expect(history.map((message) => message.content)).toEqual([
      'system prompt',
      'Goal: finish task',
      'Current state (step 2): recent',
      'Current state (step 3): now',
    ]);
    expect(history.at(-1)?.screenshot).toBeUndefined();
  });

  it('drops oldest text turns by UTF-8 bytes while retaining the goal and latest decision', () => {
    const history: ConversationMessageInterface[] = [
      { role: 'system', content: 'system prompt' },
      { role: 'user', content: 'Goal: finish task' },
      { role: 'user', content: 'Current state (step 1): old observation' },
      { role: 'assistant', content: 'Action: old decision with long explanation' },
      { role: 'user', content: 'Current state (step 2): recent observation' },
      { role: 'assistant', content: 'Action: 🚀' },
      { role: 'user', content: 'Current state (step 3): latest observation' },
    ];

    trimConversationHistory(history, 20, MAX_HISTORY_SCREENSHOT_BYTES, 4);

    expect(history.map((message) => message.content)).toEqual([
      'system prompt',
      'Goal: finish task',
      'Action: 🚀',
      'Current state (step 3): latest observation',
    ]);
    expect(Buffer.byteLength(history[2]?.content ?? '', 'utf8')).toBe(4);
  });
});
