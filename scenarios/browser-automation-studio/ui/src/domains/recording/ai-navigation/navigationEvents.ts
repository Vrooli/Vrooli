/**
 * Wire parsing and recovery projection for AI navigation.
 *
 * Keeping transport-shaped data outside owners leaves command, event, and
 * recovery admission responsible for lifecycle policy, not protocol details.
 */

import type { GetNavigationStatusResponse } from '@/api/visionNavigation';
import type {
  AINavigationAwaitingHumanEvent,
  AINavigationCompleteEvent,
  AINavigationResumedEvent,
  AINavigationState,
  AINavigationStep,
  AINavigationStepEvent,
  BrowserAction,
  TokenUsage,
} from './types';
import type { AIInferenceReceipt } from '@vrooli/proto-types/browser-automation-studio/v1/ai/ai_pb';

export const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null;

const actionTypes = new Set<BrowserAction['type']>([
  'click',
  'type',
  'scroll',
  'navigate',
  'hover',
  'select',
  'wait',
  'keypress',
  'done',
  'request_human',
  'find',
  'read',
  'evaluate',
  'tabs',
  'drag',
  'zoom',
]);

const valueBearingActionTypes = new Set<BrowserAction['type']>([
  'type',
  'find',
  'read',
  'evaluate',
  'select',
]);

const directionTypes = new Set<NonNullable<BrowserAction['direction']>>([
  'up',
  'down',
  'left',
  'right',
]);

const interventionTypes = new Set<NonNullable<BrowserAction['interventionType']>>([
  'captcha',
  'verification',
  'complex_interaction',
  'login_required',
  'other',
]);

const triggerTypes = new Set<AINavigationAwaitingHumanEvent['trigger']>([
  'programmatic',
  'ai_requested',
]);

const terminalStatusTypes = new Set<AINavigationCompleteEvent['status']>([
  'completed',
  'failed',
  'aborted',
  'max_steps_reached',
  'loop_detected',
  'awaiting_human',
]);

type NavigationEnvelope = Record<string, unknown> & {
  navigationId: string;
  sessionId: string;
};

const isNavigationEnvelope = (
  value: unknown,
  expectedType: string,
): value is NavigationEnvelope => isRecord(value)
  && value.type === expectedType
  && typeof value.navigationId === 'string'
  && typeof value.sessionId === 'string';

const stringValue = (value: unknown): string | undefined =>
  typeof value === 'string' ? value : undefined;

const numberValue = (value: unknown): number | undefined =>
  typeof value === 'number' ? value : undefined;

const booleanValue = (value: unknown): boolean | undefined =>
  typeof value === 'boolean' ? value : undefined;

const enumValue = <T extends string>(value: unknown, values: ReadonlySet<T>): T | undefined => {
  const candidate = stringValue(value);
  return candidate !== undefined && values.has(candidate as T) ? candidate as T : undefined;
};

const eventTimestamp = (value: unknown): string =>
  stringValue(value) ?? new Date().toISOString();

const assignActionField = <K extends keyof BrowserAction>(
  action: BrowserAction,
  field: K,
  value: BrowserAction[K] | undefined,
): void => {
  if (value !== undefined) action[field] = value;
};

const stringActionFields = [
  'selector',
  'text',
  'url',
  'key',
  'result',
  'reason',
  'instructions',
] as const satisfies readonly (keyof BrowserAction)[];

const parseCoordinates = (value: unknown): BrowserAction['coordinates'] | undefined => {
  if (!isRecord(value)) return undefined;
  const x = numberValue(value.x);
  const y = numberValue(value.y);
  return x !== undefined && y !== undefined ? { x, y } : undefined;
};

const parseCompleteStatus = (value: unknown): AINavigationCompleteEvent['status'] | null =>
  enumValue(value, terminalStatusTypes) ?? null;

export const parseRecoveryStatus = (value: unknown): AINavigationState['status'] | null => {
  if (value === 'navigating') return 'navigating';
  return parseCompleteStatus(value);
};

const parseTokensUsed = (value: unknown): TokenUsage => {
  const details = isRecord(value) ? value : {};
  return {
    promptTokens: numberValue(details.promptTokens) ?? 0,
    completionTokens: numberValue(details.completionTokens) ?? 0,
    totalTokens: numberValue(details.totalTokens) ?? 0,
  };
};

const exactUsageString = (value: unknown): string | undefined => {
  if (typeof value === 'string' && /^\d+$/.test(value)) return value;
  if (typeof value === 'number' && Number.isSafeInteger(value) && value >= 0) return String(value);
  return undefined;
};

const parseInferenceReceipt = (value: unknown): AIInferenceReceipt | undefined => {
  if (!isRecord(value)) return undefined;
  const rootOperationId = stringValue(value.rootOperationId);
  const source = stringValue(value.source);
  const inputTokens = exactUsageString(value.inputTokens);
  const outputTokens = exactUsageString(value.outputTokens);
  if (!rootOperationId || !source || inputTokens === undefined || outputTokens === undefined) return undefined;

  const receipt: AIInferenceReceipt = {
    $typeName: 'browser_automation_studio.v1.ai.AIInferenceReceipt',
    rootOperationId,
    billingOperationId: stringValue(value.billingOperationId) ?? '',
    source,
    provider: stringValue(value.provider) ?? '',
    model: stringValue(value.model) ?? '',
    providerCallsMade: numberValue(value.providerCallsMade) ?? 0,
    inputTokens,
    outputTokens,
    inputTokensProvenance: stringValue(value.inputTokensProvenance) ?? '',
    outputTokensProvenance: stringValue(value.outputTokensProvenance) ?? '',
    costMicros: exactUsageString(value.costMicros) ?? '',
    costProvenance: stringValue(value.costProvenance) ?? '',
    creditsCharged: exactUsageString(value.creditsCharged) ?? '',
    accountingStatus: stringValue(value.accountingStatus) ?? '',
  };
  if (isRecord(value.applied)) {
    const temperature = numberValue(value.applied.temperatureSent);
    receipt.applied = {
      $typeName: 'browser_automation_studio.v1.ai.AIInferenceAppliedSettings',
      maxOutputTokensEffective: numberValue(value.applied.maxOutputTokensEffective) ?? 0,
      maxOutputTokensSource: stringValue(value.applied.maxOutputTokensSource) ?? '',
      temperatureSent: temperature,
      temperatureSupport: stringValue(value.applied.temperatureSupport) ?? '',
    };
  }
  return receipt;
};

const applyActionDetails = (
  action: BrowserAction,
  details: Record<string, unknown>,
  valueTypes: ReadonlySet<BrowserAction['type']> = actionTypes,
): BrowserAction => {
  assignActionField(action, 'elementId', numberValue(details.elementId));
  assignActionField(action, 'coordinates', parseCoordinates(details.coordinates));

  for (const field of stringActionFields) {
    assignActionField(action, field, stringValue(details[field]));
  }

  const value = stringValue(details.value);
  if (valueTypes.has(action.type)) {
    assignActionField(action, 'value', value);
    if (action.type === 'type') assignActionField(action, 'text', value);
  }

  assignActionField(action, 'direction', enumValue(details.direction, directionTypes));
  assignActionField(action, 'success', booleanValue(details.success));
  assignActionField(action, 'interventionType', enumValue(details.interventionType, interventionTypes));
  return action;
};

const parseBrowserAction = (value: unknown): BrowserAction => {
  if (!isRecord(value) || typeof value.type !== 'string' || !actionTypes.has(value.type as BrowserAction['type'])) {
    return { type: 'wait' };
  }

  const details = isRecord(value.input)
    ? { ...value, ...value.input }
    : value;
  return applyActionDetails({ type: value.type as BrowserAction['type'] }, details);
};

const parseStepEvent = (value: unknown): AINavigationStepEvent | null => {
  if (!isNavigationEnvelope(value, 'ai_navigation_step')) return null;
  const stepNumber = numberValue(value.stepNumber);
  if (stepNumber === undefined || !Number.isInteger(stepNumber) || stepNumber < 1) return null;
  const action = parseBrowserAction(value.action);
  const reasoning = stringValue(value.reasoning) ?? '';
  const currentUrl = stringValue(value.currentUrl) ?? '';
  const goalAchieved = booleanValue(value.goalAchieved) ?? false;
  const tokensUsed = parseTokensUsed(value.tokensUsed);
  const inferenceReceipt = parseInferenceReceipt(value.inferenceReceipt);
  const durationMs = numberValue(value.durationMs) ?? 0;
  const error = stringValue(value.error);

  return {
    type: 'ai_navigation_step',
    navigationId: value.navigationId,
    sessionId: value.sessionId,
    stepNumber,
    action,
    reasoning,
    currentUrl,
    goalAchieved,
    tokensUsed,
    inferenceReceipt,
    durationMs,
    error,
    timestamp: eventTimestamp(value.timestamp),
  };
};

const parseCompleteEvent = (value: unknown): AINavigationCompleteEvent | null => {
  if (!isNavigationEnvelope(value, 'ai_navigation_complete')) return null;
  const status = parseCompleteStatus(value.status);
  if (!status) return null;
  const totalSteps = numberValue(value.totalSteps) ?? 0;
  const totalTokens = numberValue(value.totalTokens) ?? 0;
  const totalDurationMs = numberValue(value.totalDurationMs) ?? 0;
  const finalUrl = stringValue(value.finalUrl) ?? '';
  const error = stringValue(value.error);
  const summary = stringValue(value.summary);

  return {
    type: 'ai_navigation_complete',
    navigationId: value.navigationId,
    sessionId: value.sessionId,
    status,
    totalSteps,
    totalTokens,
    totalDurationMs,
    finalUrl,
    error,
    summary,
    timestamp: eventTimestamp(value.timestamp),
  };
};

const parseAwaitingHumanEvent = (value: unknown): AINavigationAwaitingHumanEvent | null => {
  if (!isNavigationEnvelope(value, 'ai_navigation_awaiting_human')) return null;
  const stepNumber = numberValue(value.stepNumber) ?? 0;
  const reason = stringValue(value.reason) ?? 'Human intervention required';
  const instructions = stringValue(value.instructions);
  const interventionType = enumValue(value.interventionType, interventionTypes) ?? 'other';
  const trigger = enumValue(value.trigger, triggerTypes) ?? 'programmatic';

  return {
    type: 'ai_navigation_awaiting_human',
    navigationId: value.navigationId,
    sessionId: value.sessionId,
    stepNumber,
    reason,
    instructions,
    interventionType,
    trigger,
    timestamp: eventTimestamp(value.timestamp),
  };
};

const parseResumedEvent = (value: unknown): AINavigationResumedEvent | null => {
  if (!isNavigationEnvelope(value, 'ai_navigation_resumed')) return null;
  return {
    type: 'ai_navigation_resumed',
    navigationId: value.navigationId,
    sessionId: value.sessionId,
    timestamp: eventTimestamp(value.timestamp),
  };
};

export type AINavigationEvent =
  | AINavigationStepEvent
  | AINavigationCompleteEvent
  | AINavigationAwaitingHumanEvent
  | AINavigationResumedEvent;

type NavigationEventParser = (value: unknown) => AINavigationEvent | null;

const navigationEventParsers: Record<string, NavigationEventParser> = {
  ai_navigation_step: parseStepEvent,
  ai_navigation_complete: parseCompleteEvent,
  ai_navigation_awaiting_human: parseAwaitingHumanEvent,
  ai_navigation_resumed: parseResumedEvent,
};

/** Dispatch wire parsing once by the declared event type. */
export const parseNavigationEvent = (value: unknown): AINavigationEvent | null => {
  if (!isRecord(value) || typeof value.type !== 'string') return null;
  if (!Object.prototype.hasOwnProperty.call(navigationEventParsers, value.type)) return null;
  return navigationEventParsers[value.type]?.(value) ?? null;
};

export const statusStepToAINavigationStep = (step: {
  index: number;
  actionType: string;
  selector: string;
  value: string;
  url: string;
  description: string;
  success: boolean;
  error: string;
  inferenceReceipt?: AIInferenceReceipt;
  at?: unknown;
}): AINavigationStep => {
  const actionType = actionTypes.has(step.actionType as BrowserAction['type'])
    ? (step.actionType as BrowserAction['type'])
    : 'wait';
  const recoveredDetails: Record<string, unknown> = {
    selector: step.selector,
    value: step.value,
    success: step.success,
  };
  if (actionType === 'navigate') recoveredDetails.url = step.url;
  const action = applyActionDetails({ type: actionType }, recoveredDetails, valueBearingActionTypes);
  const timestamp = isRecord(step.at)
    && typeof step.at.seconds === 'bigint'
    && typeof step.at.nanos === 'number'
    ? new Date(Number(step.at.seconds) * 1000 + step.at.nanos / 1_000_000)
    : new Date();

  return {
    id: `step-${step.index}`,
    stepNumber: step.index,
    action,
    reasoning: step.description,
    currentUrl: step.url,
    goalAchieved: actionType === 'done' && step.success,
    tokensUsed: { promptTokens: 0, completionTokens: 0, totalTokens: 0 },
    inferenceReceipt: step.inferenceReceipt,
    durationMs: 0,
    error: step.error || undefined,
    timestamp,
  };
};

export const mergeRecoveredSteps = (
  recoveredSteps: AINavigationStep[],
  liveSteps: AINavigationStep[],
): AINavigationStep[] => {
  const liveByNumber = new Map(liveSteps.map((step) => [step.stepNumber, step]));
  const recoveredNumbers = new Set(recoveredSteps.map((step) => step.stepNumber));
  const mergedSteps = recoveredSteps.map((recoveredStep) => {
    const liveStep = liveByNumber.get(recoveredStep.stepNumber);
    if (!liveStep) return recoveredStep;

    return {
      ...recoveredStep,
      ...liveStep,
      action: { ...recoveredStep.action, ...liveStep.action },
      reasoning: liveStep.reasoning || recoveredStep.reasoning,
      currentUrl: liveStep.currentUrl || recoveredStep.currentUrl,
      goalAchieved: liveStep.goalAchieved || recoveredStep.goalAchieved,
      tokensUsed: liveStep.tokensUsed.totalTokens > 0 ? liveStep.tokensUsed : recoveredStep.tokensUsed,
      durationMs: liveStep.durationMs > 0 ? liveStep.durationMs : recoveredStep.durationMs,
      error: liveStep.error ?? recoveredStep.error,
    };
  });

  return [
    ...mergedSteps,
    ...liveSteps.filter((step) => !recoveredNumbers.has(step.stepNumber)),
  ];
};

export const recoveredStatePatch = (
  data: GetNavigationStatusResponse,
  status: AINavigationState['status'],
  steps: AINavigationStep[],
  extractedData: Record<string, unknown> | null,
): Pick<AINavigationState, 'status' | 'steps' | 'totalTokens' | 'totalDurationMs' | 'finalUrl' | 'verifiedSuccess' | 'extractedData' | 'verificationError'> => ({
  status,
  steps,
  totalTokens: data.totalTokens,
  totalDurationMs: Number(data.totalDurationMs),
  finalUrl: data.finalUrl,
  verifiedSuccess: data.verifiedSuccess,
  extractedData,
  verificationError: data.verificationError || null,
});
