import type { BrowserAction } from '../ai-navigation/types';

export const REDACTED_ACTION_VALUE = '[REDACTED]';

const sensitiveSelectorPattern = /(?:password|passwd|passcode|secret|token|otp|one[-_ ]?time|verification[-_ ]?code|cvv|cvc|card[-_ ]?(?:number|security)|pin|credential)/i;
const sensitiveAssignmentPattern = /\b(password|passwd|passcode|secret|token|otp|one[-_ ]?time[-_ ]?code|verification[-_ ]?code|cvv|cvc|api[-_ ]?key|access[-_ ]?token|refresh[-_ ]?token|pin|credential)\b\s*[:=]\s*[^\s,;]+/gi;
const sensitiveQueryKeyPattern = /^(?:password|passwd|passcode|secret|token|otp|one[-_ ]?time(?:[-_ ]?code)?|verification[-_ ]?code|cvv|cvc|api[-_ ]?key|access[-_ ]?token|refresh[-_ ]?token|pin|credential)$/i;

export function isSensitiveAction(action: BrowserAction): boolean {
  return typeof action.selector === 'string' && sensitiveSelectorPattern.test(action.selector);
}

export function redactActionText(value: string): string {
  return value.replace(sensitiveAssignmentPattern, (_match, key: string) => `${key}=${REDACTED_ACTION_VALUE}`);
}

export function redactActionUrl(value: string): string {
  try {
    const url = new URL(value);
    const params = url.searchParams;
    let changed = false;
    for (const key of Array.from(params.keys())) {
      if (!sensitiveQueryKeyPattern.test(key)) continue;
      params.set(key, REDACTED_ACTION_VALUE);
      changed = true;
    }
    return changed ? url.toString() : value;
  } catch {
    return redactActionText(value);
  }
}

export function displayActionDetail(action: BrowserAction, field: 'value' | 'text' | 'result', value: string): string {
  if (isSensitiveAction(action) && (field === 'value' || field === 'text')) {
    return REDACTED_ACTION_VALUE;
  }
  if (isSensitiveAction(action) && field === 'result') {
    // Results are usually status text, but providers may echo the submitted
    // secret without an assignment wrapper. Preserve useful status text while
    // removing an exact value/text echo from the rendered transcript.
    const secretValues = [action.value, action.text]
      .filter((candidate): candidate is string => Boolean(candidate));
    return secretValues.reduce(
      (display, secret) => display.split(secret).join(REDACTED_ACTION_VALUE),
      value,
    );
  }
  return redactActionText(value);
}

export function displayActionPreview(action: BrowserAction): string | undefined {
  if (action.type === 'type' && action.text !== undefined) {
    return displayActionDetail(action, 'text', action.text);
  }
  if (action.type === 'select' && action.text !== undefined) {
    return displayActionDetail(action, 'text', action.text);
  }
  return undefined;
}
