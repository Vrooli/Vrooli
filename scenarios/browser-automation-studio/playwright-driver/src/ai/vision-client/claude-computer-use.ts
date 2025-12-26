/**
 * Claude Computer Use Vision Client
 *
 * STABILITY: STABLE CORE
 *
 * This module implements the VisionModelClient interface for Claude's
 * Computer Use capability. Unlike traditional vision models that parse
 * text for actions, Claude Computer Use uses native tool calls.
 *
 * Key differences from OpenRouter:
 * - Uses Anthropic's Messages API directly
 * - Uses the computer_use_20251124 tool type
 * - Actions are returned as structured tool calls
 * - Supports pixel-level coordinates natively
 *
 * Supported Claude models:
 * - claude-sonnet-4 (recommended for browser automation)
 * - claude-opus-4 (highest capability)
 */

import type {
  VisionModelClient,
  VisionAnalysisRequest,
  VisionAnalysisResponse,
  VisionModelSpec,
  TokenUsage,
} from './types';
import { VisionModelError } from './types';
import { getModelSpec } from './model-registry';
import type { BrowserAction } from '../action/types';

/**
 * Configuration for Claude Computer Use client.
 */
export interface ClaudeComputerUseClientConfig {
  /** Anthropic API key */
  apiKey: string;

  /** Model ID from registry */
  modelId: string;

  /** Base URL for Anthropic API (default: https://api.anthropic.com) */
  baseUrl?: string;

  /** Request timeout in ms (default: 60000) */
  timeoutMs?: number;

  /** Max retries on transient errors (default: 2) */
  maxRetries?: number;

  /** Display dimensions for computer tool */
  displayWidth?: number;
  displayHeight?: number;
}

/**
 * Anthropic Messages API request format.
 */
interface AnthropicRequest {
  model: string;
  max_tokens: number;
  system?: string;
  messages: AnthropicMessage[];
  tools?: AnthropicTool[];
  tool_choice?: { type: 'auto' | 'any' | 'tool'; name?: string };
}

/**
 * Anthropic message format.
 */
interface AnthropicMessage {
  role: 'user' | 'assistant';
  content: AnthropicContent[];
}

/**
 * Anthropic content block types.
 */
type AnthropicContent =
  | { type: 'text'; text: string }
  | { type: 'image'; source: { type: 'base64'; media_type: string; data: string } }
  | { type: 'tool_use'; id: string; name: string; input: Record<string, unknown> }
  | { type: 'tool_result'; tool_use_id: string; content: string };

/**
 * Anthropic tool definition.
 */
interface AnthropicTool {
  type: 'computer_20251124';
  name: 'computer';
  display_width_px: number;
  display_height_px: number;
}

/**
 * Anthropic API response format.
 */
interface AnthropicResponse {
  id: string;
  type: 'message';
  role: 'assistant';
  content: AnthropicResponseContent[];
  model: string;
  stop_reason: 'end_turn' | 'tool_use' | 'max_tokens' | 'stop_sequence' | null;
  usage: {
    input_tokens: number;
    output_tokens: number;
  };
  error?: {
    type: string;
    message: string;
  };
}

type AnthropicResponseContent =
  | { type: 'text'; text: string }
  | { type: 'tool_use'; id: string; name: string; input: ComputerToolInput };

/**
 * Computer tool input from Claude.
 */
interface ComputerToolInput {
  action:
    | 'screenshot'
    | 'key'
    | 'type'
    | 'mouse_move'
    | 'left_click'
    | 'right_click'
    | 'double_click'
    | 'scroll'
    | 'wait';
  text?: string; // For 'type' and 'key' actions
  coordinate?: [number, number]; // For mouse actions
  scroll_direction?: 'up' | 'down' | 'left' | 'right';
  scroll_amount?: number;
  duration?: number; // For 'wait' action, in seconds
}

/**
 * Claude Computer Use Vision Client implementation.
 */
export class ClaudeComputerUseClient implements VisionModelClient {
  private readonly config: Required<ClaudeComputerUseClientConfig>;
  private readonly modelSpec: VisionModelSpec;

  constructor(config: ClaudeComputerUseClientConfig) {
    this.config = {
      apiKey: config.apiKey,
      modelId: config.modelId,
      baseUrl: config.baseUrl ?? 'https://api.anthropic.com',
      timeoutMs: config.timeoutMs ?? 60000,
      maxRetries: config.maxRetries ?? 2,
      displayWidth: config.displayWidth ?? 1280,
      displayHeight: config.displayHeight ?? 800,
    };

    this.modelSpec = getModelSpec(this.config.modelId);

    if (this.modelSpec.provider !== 'anthropic') {
      throw new VisionModelError(
        `Model ${config.modelId} is not an Anthropic model. Provider: ${this.modelSpec.provider}`,
        'MODEL_UNAVAILABLE'
      );
    }

    if (!this.modelSpec.supportsComputerUse) {
      throw new VisionModelError(
        `Model ${config.modelId} does not support computer use`,
        'MODEL_UNAVAILABLE'
      );
    }
  }

  async analyze(request: VisionAnalysisRequest): Promise<VisionAnalysisResponse> {
    const anthropicRequest = this.buildRequest(request);

    let lastError: Error | null = null;
    for (let attempt = 0; attempt <= this.config.maxRetries; attempt++) {
      try {
        const response = await this.callAPI(anthropicRequest);
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
   * Build the Anthropic API request.
   */
  private buildRequest(request: VisionAnalysisRequest): AnthropicRequest {
    const messages: AnthropicMessage[] = [];

    // Build conversation history
    for (const msg of request.conversationHistory) {
      if (msg.role === 'system') continue;

      if (msg.role === 'assistant') {
        messages.push({
          role: 'assistant',
          content: [{ type: 'text', text: msg.content }],
        });
      } else if (msg.role === 'user') {
        const content: AnthropicContent[] = [];
        if (msg.screenshot) {
          content.push({
            type: 'image',
            source: {
              type: 'base64',
              media_type: this.detectImageType(msg.screenshot),
              data: msg.screenshot.toString('base64'),
            },
          });
        }
        content.push({ type: 'text', text: msg.content });
        messages.push({ role: 'user', content });
      }
    }

    // Current user message with screenshot and goal
    const userContent: AnthropicContent[] = [
      {
        type: 'image',
        source: {
          type: 'base64',
          media_type: this.detectImageType(request.screenshot),
          data: request.screenshot.toString('base64'),
        },
      },
      {
        type: 'text',
        text: this.buildUserPrompt(request),
      },
    ];

    messages.push({ role: 'user', content: userContent });

    return {
      model: this.modelSpec.apiModelId,
      max_tokens: 1024,
      system: this.buildSystemPrompt(),
      messages,
      tools: [
        {
          type: 'computer_20251124',
          name: 'computer',
          display_width_px: this.config.displayWidth,
          display_height_px: this.config.displayHeight,
        },
      ],
      tool_choice: { type: 'auto' },
    };
  }

  /**
   * Build the system prompt for Claude.
   */
  private buildSystemPrompt(): string {
    return `You are a browser automation assistant. Your task is to help the user achieve their goal by controlling a web browser.

When analyzing the screenshot:
1. Identify interactive elements (buttons, links, inputs, etc.)
2. Determine the best action to progress toward the goal
3. Use precise pixel coordinates for mouse actions
4. Explain your reasoning before taking action

If the goal has been achieved, indicate that you are done and describe what was accomplished.

Important guidelines:
- Always analyze the current page state before acting
- If an action fails or the page doesn't respond as expected, try an alternative approach
- Be careful with form submissions and irreversible actions
- If you're unsure, describe what you see and ask for clarification`;
  }

  /**
   * Build the user prompt with goal and context.
   */
  private buildUserPrompt(request: VisionAnalysisRequest): string {
    const elementInfo = request.elementLabels?.length
      ? `\n\nDetected interactive elements:\n${request.elementLabels
          .slice(0, 20)
          .map((e) => `[${e.id}] ${e.tagName}: ${e.text || e.ariaLabel || e.placeholder || '(no label)'}`)
          .join('\n')}`
      : '';

    return `Goal: ${request.goal}

Current URL: ${request.currentUrl}
${elementInfo}

Analyze the screenshot and take the next action to achieve the goal. If the goal is complete, indicate success.`;
  }

  /**
   * Call the Anthropic API.
   */
  private async callAPI(request: AnthropicRequest): Promise<AnthropicResponse> {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.config.timeoutMs);

    try {
      const response = await fetch(`${this.config.baseUrl}/v1/messages`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'x-api-key': this.config.apiKey,
          'anthropic-version': '2023-06-01',
          'anthropic-beta': 'computer-use-2025-01-24',
        },
        body: JSON.stringify(request),
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      if (!response.ok) {
        const errorBody = await response.text();
        this.handleAPIError(response.status, errorBody);
      }

      return (await response.json()) as AnthropicResponse;
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

        throw new VisionModelError(`Network error: ${error.message}`, 'NETWORK_ERROR', true, error);
      }

      throw new VisionModelError('Unknown error occurred', 'UNKNOWN', false);
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
        throw new VisionModelError(`Invalid API key: ${errorMessage}`, 'INVALID_API_KEY', false);

      case 429:
        throw new VisionModelError(`Rate limited: ${errorMessage}`, 'RATE_LIMITED', true);

      case 529:
        throw new VisionModelError(`API overloaded: ${errorMessage}`, 'RATE_LIMITED', true);

      case 400:
        if (errorMessage.includes('context') || errorMessage.includes('token')) {
          throw new VisionModelError(`Context too long: ${errorMessage}`, 'CONTEXT_TOO_LONG', false);
        }
        throw new VisionModelError(`Bad request: ${errorMessage}`, 'UNKNOWN', false);

      case 503:
      case 502:
      case 500:
        throw new VisionModelError(`Model unavailable: ${errorMessage}`, 'MODEL_UNAVAILABLE', true);

      default:
        throw new VisionModelError(`API error (${status}): ${errorMessage}`, 'UNKNOWN', status >= 500);
    }
  }

  /**
   * Parse the API response into a VisionAnalysisResponse.
   */
  private parseResponse(
    response: AnthropicResponse,
    _request: VisionAnalysisRequest
  ): VisionAnalysisResponse {
    if (response.error) {
      throw new VisionModelError(`API returned error: ${response.error.message}`, 'UNKNOWN', false);
    }

    // Extract text content (reasoning)
    let reasoning = '';
    const textBlocks = response.content.filter(
      (block): block is { type: 'text'; text: string } => block.type === 'text'
    );
    if (textBlocks.length > 0) {
      reasoning = textBlocks.map((b) => b.text).join('\n');
    }

    // Check for tool use
    const toolUseBlock = response.content.find(
      (block): block is { type: 'tool_use'; id: string; name: string; input: ComputerToolInput } =>
        block.type === 'tool_use' && block.name === 'computer'
    );

    let action: BrowserAction;
    let goalAchieved = false;

    if (toolUseBlock) {
      action = this.convertToolInputToAction(toolUseBlock.input);
    } else {
      // No tool use - Claude is done or needs clarification
      // Check if the reasoning indicates completion
      const lowerReasoning = reasoning.toLowerCase();
      const indicatesComplete =
        lowerReasoning.includes('goal achieved') ||
        lowerReasoning.includes('task complete') ||
        lowerReasoning.includes('successfully') ||
        lowerReasoning.includes('done');

      if (indicatesComplete) {
        action = {
          type: 'done',
          success: true,
          result: reasoning.substring(0, 200),
        };
        goalAchieved = true;
      } else {
        // Default to wait if no action specified
        action = {
          type: 'wait',
          ms: 1000,
        };
      }
    }

    // Check if the action is done
    if (action.type === 'done') {
      goalAchieved = action.success;
    }

    const tokensUsed: TokenUsage = {
      promptTokens: response.usage.input_tokens,
      completionTokens: response.usage.output_tokens,
      totalTokens: response.usage.input_tokens + response.usage.output_tokens,
    };

    return {
      action,
      reasoning,
      goalAchieved,
      confidence: this.estimateConfidence(action, reasoning),
      tokensUsed,
      rawResponse: JSON.stringify(response.content),
    };
  }

  /**
   * Convert Claude's computer tool input to our BrowserAction type.
   */
  private convertToolInputToAction(input: ComputerToolInput): BrowserAction {
    switch (input.action) {
      case 'left_click':
        if (input.coordinate) {
          return {
            type: 'click',
            coordinates: { x: input.coordinate[0], y: input.coordinate[1] },
            variant: 'left',
          };
        }
        return { type: 'click', variant: 'left' };

      case 'right_click':
        if (input.coordinate) {
          return {
            type: 'click',
            coordinates: { x: input.coordinate[0], y: input.coordinate[1] },
            variant: 'right',
          };
        }
        return { type: 'click', variant: 'right' };

      case 'double_click':
        if (input.coordinate) {
          return {
            type: 'click',
            coordinates: { x: input.coordinate[0], y: input.coordinate[1] },
            variant: 'double',
          };
        }
        return { type: 'click', variant: 'double' };

      case 'mouse_move':
        if (input.coordinate) {
          return {
            type: 'hover',
            coordinates: { x: input.coordinate[0], y: input.coordinate[1] },
          };
        }
        return { type: 'hover' };

      case 'type':
        return {
          type: 'type',
          text: input.text ?? '',
        };

      case 'key':
        return {
          type: 'keypress',
          key: input.text ?? 'Enter',
        };

      case 'scroll':
        return {
          type: 'scroll',
          direction: input.scroll_direction ?? 'down',
          amount: input.scroll_amount ?? 300,
        };

      case 'wait':
        return {
          type: 'wait',
          ms: (input.duration ?? 1) * 1000,
        };

      case 'screenshot':
        // Screenshot action means Claude wants to see the result
        // We'll treat this as a wait since we always capture after each action
        return {
          type: 'wait',
          ms: 500,
        };

      default:
        return {
          type: 'wait',
          ms: 1000,
        };
    }
  }

  /**
   * Detect image MIME type from buffer.
   */
  private detectImageType(buffer: Buffer): string {
    const isJPEG = buffer[0] === 0xff && buffer[1] === 0xd8;
    const isPNG = buffer[0] === 0x89 && buffer[1] === 0x50;
    const isGIF = buffer[0] === 0x47 && buffer[1] === 0x49;
    const isWebP = buffer[8] === 0x57 && buffer[9] === 0x45 && buffer[10] === 0x42;

    if (isJPEG) return 'image/jpeg';
    if (isPNG) return 'image/png';
    if (isGIF) return 'image/gif';
    if (isWebP) return 'image/webp';
    return 'image/png'; // Default
  }

  /**
   * Estimate confidence based on action and reasoning.
   */
  private estimateConfidence(action: BrowserAction, reasoning: string): number {
    let confidence = 0.85;

    if (action.type === 'done') {
      confidence = action.success ? 0.95 : 0.8;
    }

    if (action.type === 'click' && 'coordinates' in action && action.coordinates) {
      confidence = 0.9;
    }

    if (action.type === 'scroll' || action.type === 'wait') {
      confidence = 0.7;
    }

    if (reasoning.length < 50) {
      confidence *= 0.9;
    }

    if (reasoning.length > 200) {
      confidence *= 1.05;
    }

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
 * Factory function to create a Claude Computer Use client.
 */
export function createClaudeComputerUseClient(
  config: ClaudeComputerUseClientConfig
): ClaudeComputerUseClient {
  return new ClaudeComputerUseClient(config);
}
