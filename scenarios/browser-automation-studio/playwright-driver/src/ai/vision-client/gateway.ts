/**
 * AI Gateway vision client.
 *
 * This is the only hosted/local multimodal client used by the Playwright
 * navigator. It sends provider-neutral inference intent to AI Gateway over
 * Connect's JSON protocol. Provider URLs, credentials, concrete model IDs,
 * retries, and usage accounting remain gateway/resource concerns.
 */

import type {
  ConversationMessage,
  GatewayProfile,
  TokenUsage,
  VisionAnalysisRequest,
  VisionAnalysisResponse,
  VisionModelClient,
  VisionModelSpec,
} from './types';
import { VisionModelError } from './types';
import { parseLLMResponse } from '../action/parser';
import type { BrowserAction } from '../action/types';

const INFERENCE_PROCEDURE =
  '/vrooli.ai_gateway.v1.inference.InferenceService/Run';
const DEFAULT_TIMEOUT_MS = 120_000;
const DEFAULT_ROLE = 'extract.structured';

export interface AIGatewayVisionClientConfig {
  /** AI Gateway API base URL. The managed BAS sidecar receives this via env. */
  gatewayUrl?: string;
  /** Provider-neutral routing profile. */
  profile?: GatewayProfile;
  /** Request timeout in milliseconds. */
  timeoutMs?: number;
}

interface GatewayUsage {
  inputTokens?: number | string;
  outputTokens?: number | string;
}

interface GatewayResponse {
  valueJson?: string;
  usage?: GatewayUsage;
  provider?: string;
  model?: string;
  validated?: boolean;
  error?: {
    code?: string;
    message?: string;
  };
}

interface GatewayTurn {
  role: 'user' | 'assistant';
  text: string;
  attachments?: GatewayAttachment[];
}

interface GatewayAttachment {
  modality: 'MODALITY_IMAGE';
  mediaType: string;
  width: number;
  height: number;
  bytes: number;
  inlineBytes: string;
}

interface GatewayRunRequest {
  source: string;
  schemaJson: string;
  instruction: string;
  role: string;
  turns: GatewayTurn[];
  profile: 'PROFILE_LOCAL_FIRST' | 'PROFILE_REMOTE_ONLY';
}

function routeSpec(profile: GatewayProfile): VisionModelSpec {
  return {
    id: profile,
    displayName: profile === 'local_first' ? 'Local-first vision' : 'Hosted vision',
    provider: 'ai-gateway',
    supportsComputerUse: false,
    supportsElementLabels: true,
    recommended: profile === 'local_first',
    tier: profile === 'local_first' ? 'local' : 'remote',
  };
}

export function normalizeGatewayProfile(value?: string): GatewayProfile {
  const normalized = (value ?? '').trim().toLowerCase().replace(/-/g, '_');
  if (!normalized || normalized === 'local_first') {
    return 'local_first';
  }
  if (normalized === 'remote_only' || normalized === 'hosted') {
    return 'remote_only';
  }
  throw new VisionModelError(`Unsupported AI Gateway profile: ${value}`, 'MODEL_UNAVAILABLE');
}

function parseUsage(value: number | string | undefined): number {
  const parsed = typeof value === 'string' ? Number.parseInt(value, 10) : value ?? 0;
  return Number.isFinite(parsed) && parsed >= 0 ? parsed : 0;
}

function detectMediaType(image: Buffer): string {
  if (image.length >= 2 && image[0] === 0xff && image[1] === 0xd8) return 'image/jpeg';
  if (image.length >= 8 && image[0] === 0x89 && image[1] === 0x50) return 'image/png';
  if (image.subarray(0, 5).toString('ascii') === '<?xml' || image.subarray(0, 4).toString('ascii') === '<svg') {
    return 'image/svg+xml';
  }
  return 'image/png';
}

function dimensionsFromPNG(image: Buffer): { width: number; height: number } {
  if (image.length >= 24 && image.readUInt32BE(0) === 0x89504e47) {
    return { width: image.readUInt32BE(16), height: image.readUInt32BE(20) };
  }
  return { width: 0, height: 0 };
}

function dimensionsFromJPEG(image: Buffer): { width: number; height: number } {
  if (image.length < 4 || image[0] !== 0xff || image[1] !== 0xd8) {
    return { width: 0, height: 0 };
  }

  let offset = 2;
  while (offset + 3 < image.length) {
    if (image[offset] !== 0xff) {
      offset += 1;
      continue;
    }
    while (offset < image.length && image[offset] === 0xff) offset += 1;
    if (offset >= image.length) break;

    const marker = image[offset++]!;
    if (marker === 0xd8 || marker === 0xd9) continue;
    if (marker === 0xda || offset + 1 >= image.length) break;

    const segmentLength = image.readUInt16BE(offset);
    if (segmentLength < 2 || offset + segmentLength > image.length) break;

    const isStartOfFrame =
      (marker >= 0xc0 && marker <= 0xc3) ||
      (marker >= 0xc5 && marker <= 0xc7) ||
      (marker >= 0xc9 && marker <= 0xcb) ||
      (marker >= 0xcd && marker <= 0xcf);
    if (isStartOfFrame && segmentLength >= 7) {
      return {
        height: image.readUInt16BE(offset + 3),
        width: image.readUInt16BE(offset + 5),
      };
    }
    offset += segmentLength;
  }
  return { width: 0, height: 0 };
}

function dimensionsFromImage(image: Buffer): { width: number; height: number } {
  const png = dimensionsFromPNG(image);
  return png.width > 0 && png.height > 0 ? png : dimensionsFromJPEG(image);
}

function imageAttachment(image: Buffer): GatewayAttachment {
  const dimensions = dimensionsFromImage(image);
  if (dimensions.width <= 0 || dimensions.height <= 0) {
    throw new VisionModelError('Screenshot dimensions could not be read', 'PARSE_ERROR');
  }
  return {
    modality: 'MODALITY_IMAGE',
    mediaType: detectMediaType(image),
    width: dimensions.width,
    height: dimensions.height,
    bytes: image.byteLength,
    inlineBytes: image.toString('base64'),
  };
}

const ACTION_SCHEMA = JSON.stringify({
  type: 'object',
  required: ['action', 'reasoning', 'goalAchieved', 'confidence'],
  properties: {
    action: {
      type: 'object',
      required: ['type'],
      properties: {
        type: { type: 'string' },
        elementId: { type: 'integer' },
        coordinates: {
          type: 'object',
          required: ['x', 'y'],
          properties: { x: { type: 'number' }, y: { type: 'number' } },
        },
        text: { type: 'string' },
        direction: { type: 'string' },
        amount: { type: 'number' },
        url: { type: 'string' },
        key: { type: 'string' },
        result: { type: 'string' },
        success: { type: 'boolean' },
        reason: { type: 'string' },
        instructions: { type: 'string' },
        interventionType: { type: 'string' },
      },
    },
    reasoning: { type: 'string' },
    goalAchieved: { type: 'boolean' },
    confidence: { type: 'number', minimum: 0, maximum: 1 },
  },
});

function gatewayInstruction(): string {
  return `Return one JSON object matching the supplied schema. The action object must use one of the browser action types: click, type, scroll, navigate, hover, select, wait, keypress, done, or request_human. Use elementId for numbered labels when possible. Include concise reasoning, goalAchieved, and confidence. Do not return Markdown, prose outside the JSON object, or provider-specific fields.`;
}

function messageText(message: ConversationMessage): string {
  return message.content.trim();
}

function buildTurns(request: VisionAnalysisRequest): GatewayTurn[] {
  const turns: GatewayTurn[] = [];
  for (const message of request.conversationHistory) {
    if (message.role === 'system') continue;
    const turn: GatewayTurn = {
      role: message.role === 'assistant' ? 'assistant' : 'user',
      text: messageText(message),
    };
    if (message.role === 'user' && message.screenshot) {
      turn.attachments = [imageAttachment(message.screenshot)];
    }
    turns.push(turn);
  }

  // The current frame is always attached to the current user turn. This is
  // deliberately separate from the history so callers can retain prior
  // frames without flattening the ordered turn contract.
  turns.push({
    role: 'user',
    text: `Goal: ${request.goal}\nCurrent URL: ${request.currentUrl}`,
    attachments: [imageAttachment(request.screenshot)],
  });
  return turns;
}

function decodeValue(valueJson: string): {
  action: BrowserAction;
  reasoning: string;
  goalAchieved: boolean;
  confidence: number;
} {
  let parsed: unknown;
  try {
    parsed = JSON.parse(valueJson);
  } catch (error) {
    throw new VisionModelError(`Gateway returned invalid JSON: ${String(error)}`, 'PARSE_ERROR');
  }

  const record = parsed && typeof parsed === 'object' ? parsed as Record<string, unknown> : null;
  const candidate = record && 'action' in record ? record.action : parsed;
  if (!candidate || typeof candidate !== 'object') {
    throw new VisionModelError('Gateway response did not contain an action object', 'PARSE_ERROR');
  }

  // The envelope owns goalAchieved, while the legacy browser action contract
  // requires done.success. Some otherwise-valid structured responses omit the
  // duplicated action field, so bridge the two contracts here rather than
  // rejecting a response that the gateway already validated.
  const envelopeGoalAchieved = record && typeof record.goalAchieved === 'boolean'
    ? record.goalAchieved
    : false;
  const actionRecord = candidate as Record<string, unknown>;
  const normalizedCandidate = actionRecord.type === 'done' && typeof actionRecord.success !== 'boolean'
    ? { ...actionRecord, success: envelopeGoalAchieved }
    : candidate;

  let action: BrowserAction;
  try {
    action = parseLLMResponse(JSON.stringify(normalizedCandidate));
  } catch (error) {
    throw new VisionModelError(`Gateway action was invalid: ${String(error)}`, 'PARSE_ERROR');
  }

  const reasoning = record && typeof record.reasoning === 'string' ? record.reasoning : '';
  const goalAchieved = record && typeof record.goalAchieved === 'boolean'
    ? record.goalAchieved
    : action.type === 'done' && action.success;
  const confidence = record && typeof record.confidence === 'number'
    ? Math.max(0, Math.min(1, record.confidence))
    : action.type === 'done' && action.success ? 0.95 : 0.75;
  return { action, reasoning, goalAchieved, confidence };
}

export class AIGatewayVisionClient implements VisionModelClient {
  private readonly gatewayUrl: string;
  private readonly profile: GatewayProfile;
  private readonly timeoutMs: number;

  constructor(config: AIGatewayVisionClientConfig = {}) {
    this.gatewayUrl = (config.gatewayUrl ?? process.env.AI_GATEWAY_URL ?? '').replace(/\/$/, '');
    if (!this.gatewayUrl) {
      throw new VisionModelError('AI_GATEWAY_URL is not configured', 'MODEL_UNAVAILABLE');
    }
    this.profile = config.profile ?? 'local_first';
    this.timeoutMs = config.timeoutMs ?? DEFAULT_TIMEOUT_MS;
  }

  async analyze(request: VisionAnalysisRequest): Promise<VisionAnalysisResponse> {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), this.timeoutMs);
    const body: GatewayRunRequest = {
      source: `browser navigation at ${request.currentUrl}`,
      schemaJson: ACTION_SCHEMA,
      instruction: gatewayInstruction(),
      role: DEFAULT_ROLE,
      turns: buildTurns(request),
      profile: this.profile === 'remote_only' ? 'PROFILE_REMOTE_ONLY' : 'PROFILE_LOCAL_FIRST',
    };

    try {
      const response = await fetch(`${this.gatewayUrl}${INFERENCE_PROCEDURE}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
        signal: controller.signal,
      });
      const payload = await response.json() as GatewayResponse;
      if (!response.ok) {
        throw new VisionModelError(`AI Gateway request failed (${response.status})`, 'NETWORK_ERROR', response.status >= 500);
      }
      if (payload.error?.message) {
        throw new VisionModelError(payload.error.message, 'MODEL_UNAVAILABLE', true);
      }
      if (!payload.validated || !payload.valueJson) {
        throw new VisionModelError('AI Gateway rejected the browser action response', 'PARSE_ERROR');
      }
      const value = decodeValue(payload.valueJson);
      const tokensUsed: TokenUsage = {
        promptTokens: parseUsage(payload.usage?.inputTokens),
        completionTokens: parseUsage(payload.usage?.outputTokens),
        totalTokens: parseUsage(payload.usage?.inputTokens) + parseUsage(payload.usage?.outputTokens),
      };
      return { ...value, tokensUsed, rawResponse: payload.valueJson };
    } catch (error) {
      if (error instanceof VisionModelError) throw error;
      if (error instanceof Error && error.name === 'AbortError') {
        throw new VisionModelError(`AI Gateway request timed out after ${this.timeoutMs}ms`, 'NETWORK_ERROR', true);
      }
      throw new VisionModelError(`AI Gateway request failed: ${String(error)}`, 'NETWORK_ERROR', true);
    } finally {
      clearTimeout(timeout);
    }
  }

  getModelSpec(): VisionModelSpec {
    return routeSpec(this.profile);
  }
}

export function createAIGatewayVisionClient(config: AIGatewayVisionClientConfig = {}): AIGatewayVisionClient {
  return new AIGatewayVisionClient(config);
}
