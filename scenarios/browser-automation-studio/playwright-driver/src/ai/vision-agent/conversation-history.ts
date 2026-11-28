import type { ConversationMessageInterface } from './types';

/** Keep prior multimodal context bounded; the current screenshot is sent separately. */
export const MAX_HISTORY_SCREENSHOT_BYTES = 2 * 1024 * 1024;
/** Bound older textual turns independently of the prompt and current observation. */
export const MAX_HISTORY_TEXT_BYTES = 32 * 1024;

/**
 * Bound retained model context by message count and image bytes. Preserve the
 * system prompt, goal, and current textual observation. Screenshots are removed
 * from oldest retained messages first; the current screenshot is already a
 * separate analyze() argument and must not be duplicated in history.
 */
export function trimConversationHistory(
  history: ConversationMessageInterface[],
  maxMessages: number,
  maxScreenshotBytes = MAX_HISTORY_SCREENSHOT_BYTES,
  maxTextBytes = MAX_HISTORY_TEXT_BYTES
): void {
  const protectedEnd = findProtectedPrefixEnd(history);
  const current = history[history.length - 1];
  if (current?.role === 'user' && current.content.startsWith('Current state (step ')) {
    current.screenshot = undefined;
  }

  let retainedScreenshotBytes = history.reduce(
    (total, message) => total + (message.screenshot?.byteLength ?? 0),
    0
  );
  for (
    let index = protectedEnd;
    index < history.length && retainedScreenshotBytes > maxScreenshotBytes;
    index += 1
  ) {
    const message = history[index];
    if (!message?.screenshot) continue;
    retainedScreenshotBytes -= message.screenshot.byteLength;
    message.screenshot = undefined;
  }

  // Keep the system prompt, goal, latest observation, and the immediately
  // preceding model decision intact. Older textual turns are expendable once
  // their UTF-8 payload exceeds the history budget.
  const latest = history.length - 1;
  const protectedText = new Set<number>();
  for (let index = 0; index < protectedEnd; index += 1) protectedText.add(index);
  if (latest >= 0) protectedText.add(latest);
  if (latest > protectedEnd && history[latest - 1]?.role === 'assistant') {
    protectedText.add(latest - 1);
  }
  let retainedTextBytes = history.reduce(
    (total, message, index) =>
      total + (protectedText.has(index) ? 0 : Buffer.byteLength(message.content, 'utf8')),
    0
  );
  for (let index = protectedEnd; index < history.length && retainedTextBytes > maxTextBytes; ) {
    if (protectedText.has(index)) {
      index += 1;
      continue;
    }
    const [removed] = history.splice(index, 1);
    if (removed) retainedTextBytes -= Buffer.byteLength(removed.content, 'utf8');
    for (const protectedIndex of [...protectedText]) {
      protectedText.delete(protectedIndex);
      protectedText.add(protectedIndex > index ? protectedIndex - 1 : protectedIndex);
    }
  }

  while (history.length > maxMessages && history.length > protectedEnd + 1) {
    history.splice(protectedEnd, 1);
  }
}

function findProtectedPrefixEnd(history: ConversationMessageInterface[]): number {
  let end = 0;
  for (let index = 0; index < history.length && index < 3; index += 1) {
    const entry = history[index];
    if (!entry) continue;
    if (entry.role === 'system') end = index + 1;
    else if (index === 1 && entry.role === 'user' && entry.content.startsWith('Goal:'))
      end = index + 1;
  }
  return end;
}
