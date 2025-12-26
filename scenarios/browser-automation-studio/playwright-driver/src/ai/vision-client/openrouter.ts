/**
 * OpenRouter Vision Client
 *
 * STABILITY: STABLE CORE
 *
 * This module implements the VisionModelClient interface for OpenRouter.
 * OpenRouter provides access to multiple vision models including:
 * - Qwen3-VL-30B (budget tier, excellent for browser automation)
 * - GPT-4o (standard tier, high accuracy)
 * - GPT-4o-mini (budget tier, faster)
 *
 * The client handles:
 * - Image encoding (base64)
 * - Prompt formatting with element labels
 * - Response parsing using the action parser
 * - Token usage tracking
 * - Error handling and retries
 */

import type {
  VisionModelClient,
  VisionAnalysisRequest,
  VisionAnalysisResponse,
  VisionModelSpec,
  TokenUsage,
  ConversationMessage,
} from './types';
import { VisionModelError } from './types';
import { getModelSpec } from './model-registry';
import { generateSystemPrompt, generateUserPrompt } from './prompts';
import { parseLLMResponse, extractReasoning, ActionParseError } from '../action/parser';

/**
 * Configuration for OpenRouter client.
 */
export interface OpenRouterClientConfig {
  /** OpenRouter API key */
  apiKey: string;

  /** Model ID from registry */
  modelId: string;

  /** Base URL for OpenRouter API (default: https://openrouter.ai/api/v1) */
  baseUrl?: string;

  /** Request timeout in ms (default: 60000) */
  timeoutMs?: number;

  /** Max retries on transient errors (default: 2) */
  maxRetries?: number;

  /** Custom HTTP referrer for OpenRouter tracking */
  httpReferer?: string;

  /** Custom app title for OpenRouter tracking */
  appTitle?: string;
}

/**
 * OpenRouter API message format.
 */
interface OpenRouterMessage {
  role: 'system' | 'user' | 'assistant';
  content: string | OpenRouterContent[];
}

/**
 * OpenRouter content block (for multimodal messages).
 */
type OpenRouterContent =
  | { type: 'text'; text: string }
  | { type: 'image_url'; image_url: { url: string; detail?: 'low' | 'high' | 'auto' } };

/**
 * OpenRouter API response.
 */
interface OpenRouterResponse {
  id: string;
  model: string;
  choices: {
    index: number;
    message: {
      role: 'assistant';
      content: string;
    };
    finish_reason: 'stop' | 'length' | 'content_filter' | 'tool_calls' | null;
  }[];
  usage?: {
    prompt_tokens: number;
    completion_tokens: number;
    total_tokens: number;
  };
  error?: {
    message: string;
    code: number;
    type: string;
  };
}

/**
 * OpenRouter Vision Client implementation.
 */
export class OpenRouterVisionClient implements VisionModelClient {
  private readonly config: Required<OpenRouterClientConfig>;
  private readonly modelSpec: VisionModelSpec;

  constructor(config: OpenRouterClientConfig) {
    this.config = {
      apiKey: config.apiKey,
      modelId: config.modelId,
      baseUrl: config.baseUrl ?? 'https://openrouter.ai/api/v1',
      timeoutMs: config.timeoutMs ?? 60000,
      maxRetries: config.maxRetries ?? 2,
      httpReferer: config.httpReferer ?? 'https://vrooli.com',
      appTitle: config.appTitle ?? 'Browser Automation Studio',
    };

    this.modelSpec = getModelSpec(this.config.modelId);

    if (this.modelSpec.provider !== 'openrouter') {
      throw new VisionModelError(
        `Model ${config.modelId} is not an OpenRouter model. Provider: ${this.modelSpec.provider}`,
        'MODEL_UNAVAILABLE'
      );
    }
  }

  async analyze(request: VisionAnalysisRequest): Promise<VisionAnalysisResponse> {
    const messages = this.buildMessages(request);

    let lastError: Error | null = null;
    for (let attempt = 0; attempt <= this.config.maxRetries; attempt++) {
      try {
        const response = await this.callAPI(messages);
        return this.parseResponse(response, request);
      } catch (error) {
        lastError = error instanceof Error ? error : new Error(String(error));

        // Don't retry non-retryable errors
        if (error instanceof VisionModelError && !error.retryable) {
          throw error;
        }

        // Don't retry on last attempt
        if (attempt === this.config.maxRetries) {
          break;
        }

        // Exponential backoff
        await this.sleep(Math.pow(2, attempt) * 1000);
      }
    }

    // If the last error was a VisionModelError, re-throw it with additional context
    if (lastError instanceof VisionModelError) {
      throw lastError;
    }

    throw new VisionModelError(
      `Failed after ${this.config.maxRetries + 1} attempts: ${lastError?.message}`,
      'NETWORK_ERROR',
      false,
      lastError ?? undefined
    );
  }

  getModelSpec(): VisionModelSpec {
    return this.modelSpec;
  }

  /**
   * Build messages array for the API call.
   */
  private buildMessages(request: VisionAnalysisRequest): OpenRouterMessage[] {
    const messages: OpenRouterMessage[] = [];

    // System message
    messages.push({
      role: 'system',
      content: generateSystemPrompt(),
    });

    // Add conversation history
    for (const msg of request.conversationHistory) {
      if (msg.role === 'system') continue; // Skip system messages in history

      if (msg.role === 'assistant') {
        messages.push({
          role: 'assistant',
          content: msg.content,
        });
      } else if (msg.role === 'user') {
        // User messages may include screenshots
        if (msg.screenshot) {
          messages.push({
            role: 'user',
            content: [
              {
                type: 'image_url',
                image_url: {
                  url: this.encodeImage(msg.screenshot),
                  detail: 'high',
                },
              },
              { type: 'text', text: msg.content },
            ],
          });
        } else {
          messages.push({
            role: 'user',
            content: msg.content,
          });
        }
      }
    }

    // Current user message with screenshot
    const stepNumber = request.conversationHistory.filter(m => m.role === 'user').length + 1;
    const previousActions = request.conversationHistory
      .filter((m): m is ConversationMessage & { role: 'assistant' } => m.role === 'assistant')
      .map(m => {
        // Try to extract the action from assistant messages
        const actionMatch = m.content.match(/ACTION:\s*(\w+\([^)]*\))/i);
        return actionMatch ? actionMatch[1] : m.content.substring(0, 100);
      });

    const userPrompt = generateUserPrompt({
      goal: request.goal,
      currentUrl: request.currentUrl,
      elementLabels: request.elementLabels ?? [],
      stepNumber,
      previousActions: previousActions.length > 0 ? previousActions : undefined,
    });

    const userContent: OpenRouterContent[] = [
      {
        type: 'image_url',
        image_url: {
          url: this.encodeImage(request.screenshot),
          detail: 'high',
        },
      },
      { type: 'text', text: userPrompt },
    ];

    messages.push({
      role: 'user',
      content: userContent,
    });

    return messages;
  }

  /**
   * Call the OpenRouter API.
   */
  private async callAPI(messages: OpenRouterMessage[]): Promise<OpenRouterResponse> {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.config.timeoutMs);

    try {
      const response = await fetch(`${this.config.baseUrl}/chat/completions`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${this.config.apiKey}`,
          'HTTP-Referer': this.config.httpReferer,
          'X-Title': this.config.appTitle,
        },
        body: JSON.stringify({
          model: this.modelSpec.apiModelId,
          messages,
          max_tokens: 1024,
          temperature: 0.1, // Low temperature for deterministic actions
        }),
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      if (!response.ok) {
        const errorBody = await response.text();
        this.handleAPIError(response.status, errorBody);
      }

      return await response.json() as OpenRouterResponse;
    } catch (error) {
      clearTimeout(timeoutId);

      if (error instanceof VisionModelError) {
        throw error;
      }

      if (error instanceof Error) {
        if (error.name === 'AbortError') {
          throw new VisionModelError(
            `Request timed out after ${this.config.timeoutMs}ms`,
            'NETWORK_ERROR',
            true
          );
        }

        throw new VisionModelError(
          `Network error: ${error.message}`,
          'NETWORK_ERROR',
          true,
          error
        );
      }

      throw new VisionModelError(
        'Unknown error occurred',
        'UNKNOWN',
        false
      );
    }
  }

  /**
   * Handle API error responses.
   */
  private handleAPIError(status: number, body: string): never {
    let errorMessage = body;
    try {
      const parsed = JSON.parse(body);
      errorMessage = parsed.error?.message ?? parsed.message ?? body;
    } catch {
      // Use raw body if not JSON
    }

    switch (status) {
      case 401:
        throw new VisionModelError(
          `Invalid API key: ${errorMessage}`,
          'INVALID_API_KEY',
          false
        );

      case 429:
        throw new VisionModelError(
          `Rate limited: ${errorMessage}`,
          'RATE_LIMITED',
          true
        );

      case 402:
        throw new VisionModelError(
          `Quota exceeded: ${errorMessage}`,
          'QUOTA_EXCEEDED',
          false
        );

      case 503:
      case 502:
      case 500:
        throw new VisionModelError(
          `Model unavailable: ${errorMessage}`,
          'MODEL_UNAVAILABLE',
          true
        );

      default:
        throw new VisionModelError(
          `API error (${status}): ${errorMessage}`,
          'UNKNOWN',
          status >= 500
        );
    }
  }

  /**
   * Parse the API response into a VisionAnalysisResponse.
   */
  private parseResponse(
    response: OpenRouterResponse,
    request: VisionAnalysisRequest
  ): VisionAnalysisResponse {
    if (response.error) {
      throw new VisionModelError(
        `API returned error: ${response.error.message}`,
        'UNKNOWN',
        false
      );
    }

    const choice = response.choices[0];
    if (!choice) {
      throw new VisionModelError(
        'No response from model',
        'PARSE_ERROR',
        false
      );
    }

    const rawContent = choice.message.content;

    // Parse the action from the response
    let action;
    try {
      action = parseLLMResponse(rawContent);
    } catch (error) {
      if (error instanceof ActionParseError) {
        throw new VisionModelError(
          `Failed to parse action: ${error.message}`,
          'PARSE_ERROR',
          false,
          error
        );
      }
      throw error;
    }

    // Extract reasoning (text before the action)
    const reasoning = extractReasoning(rawContent);

    // Determine if goal is achieved
    const goalAchieved = action.type === 'done' && action.success;

    // Calculate token usage
    const tokensUsed: TokenUsage = response.usage
      ? {
          promptTokens: response.usage.prompt_tokens,
          completionTokens: response.usage.completion_tokens,
          totalTokens: response.usage.total_tokens,
        }
      : {
          // Estimate if not provided
          promptTokens: this.estimateTokens(request),
          completionTokens: Math.ceil(rawContent.length / 4),
          totalTokens: 0, // Will be calculated
        };

    if (!response.usage) {
      tokensUsed.totalTokens = tokensUsed.promptTokens + tokensUsed.completionTokens;
    }

    // Estimate confidence based on action type and reasoning
    const confidence = this.estimateConfidence(action, reasoning);

    return {
      action,
      reasoning,
      goalAchieved,
      confidence,
      tokensUsed,
      rawResponse: rawContent,
    };
  }

  /**
   * Encode an image buffer to a base64 data URL.
   */
  private encodeImage(buffer: Buffer): string {
    // Detect image type from magic bytes
    const isJPEG = buffer[0] === 0xff && buffer[1] === 0xd8;
    const isPNG = buffer[0] === 0x89 && buffer[1] === 0x50;

    const mimeType = isJPEG ? 'image/jpeg' : isPNG ? 'image/png' : 'image/png';
    const base64 = buffer.toString('base64');

    return `data:${mimeType};base64,${base64}`;
  }

  /**
   * Estimate token count for a request.
   */
  private estimateTokens(request: VisionAnalysisRequest): number {
    // Rough estimation:
    // - System prompt: ~800 tokens
    // - Image: ~1000-2000 tokens depending on detail
    // - Element labels: ~20 tokens per element
    // - Goal and URL: ~50 tokens
    // - History: varies

    const systemTokens = 800;
    const imageTokens = 1500;
    const elementTokens = (request.elementLabels?.length ?? 0) * 20;
    const textTokens = 50 + (request.goal.length / 4);
    const historyTokens = request.conversationHistory.reduce((acc, msg) => {
      return acc + Math.ceil(msg.content.length / 4) + (msg.screenshot ? 1500 : 0);
    }, 0);

    return Math.ceil(systemTokens + imageTokens + elementTokens + textTokens + historyTokens);
  }

  /**
   * Estimate confidence based on action and reasoning.
   */
  private estimateConfidence(action: VisionAnalysisResponse['action'], reasoning: string): number {
    // Base confidence
    let confidence = 0.8;

    // Done actions are usually more certain
    if (action.type === 'done') {
      confidence = action.success ? 0.95 : 0.85;
    }

    // Click and type with element IDs are more certain
    if ((action.type === 'click' || action.type === 'type') && 'elementId' in action && action.elementId) {
      confidence = 0.9;
    }

    // Scroll and wait are uncertain (exploration)
    if (action.type === 'scroll' || action.type === 'wait') {
      confidence = 0.6;
    }

    // Reduce confidence for short reasoning (may indicate uncertainty)
    if (reasoning.length < 50) {
      confidence *= 0.9;
    }

    // Increase confidence for detailed reasoning
    if (reasoning.length > 200) {
      confidence *= 1.05;
    }

    // Clamp to [0, 1]
    return Math.min(1, Math.max(0, confidence));
  }

  /**
   * Sleep for a given number of milliseconds.
   */
  private sleep(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }
}

/**
 * Factory function to create an OpenRouter client.
 */
export function createOpenRouterClient(config: OpenRouterClientConfig): OpenRouterVisionClient {
  return new OpenRouterVisionClient(config);
}
