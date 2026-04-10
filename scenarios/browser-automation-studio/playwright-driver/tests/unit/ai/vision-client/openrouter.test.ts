/**
 * OpenRouter Vision Client Tests
 *
 * Tests the OpenRouterVisionClient implementation with mocked fetch.
 */

import {
  OpenRouterVisionClient,
  createOpenRouterClient,
} from '../../../../src/ai/vision-client/openrouter';
import { VisionModelError } from '../../../../src/ai/vision-client/types';

type OpenRouterMessage = {
  role: string;
  content: unknown;
};

type OpenRouterContent =
  | { type: 'text'; text: string }
  | { type: 'image_url'; image_url: { url: string } };

// Mock fetch globally
const mockFetch = jest.fn<Promise<Response>, [RequestInfo | URL, RequestInit?]>();
global.fetch = mockFetch;

const createJsonResponse = (value: unknown, init?: ResponseInit): Response =>
  new Response(JSON.stringify(value), {
    status: init?.status ?? 200,
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
  });

const createTextResponse = (value: string, status: number): Response =>
  new Response(value, { status });

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null;

const getRequestOptions = (): RequestInit => {
  const call = mockFetch.mock.calls[0];
  if (!call) {
    throw new Error('Expected fetch to have been called');
  }
  const options = call[1];
  if (!options) {
    throw new Error('Expected fetch to have been called with options');
  }
  return options;
};

const getRequestBody = (): Record<string, unknown> => {
  const options = getRequestOptions();
  const body = options.body;
  if (typeof body !== 'string') {
    throw new Error('Expected request body to be a JSON string');
  }
  const parsed: unknown = JSON.parse(body);
  if (!isRecord(parsed)) {
    throw new Error('Expected JSON body to be an object');
  }
  return parsed;
};

const isMessage = (value: unknown): value is OpenRouterMessage =>
  isRecord(value) && typeof value.role === 'string' && 'content' in value;

const getMessages = (body: Record<string, unknown>): OpenRouterMessage[] => {
  const messages = body.messages;
  if (!Array.isArray(messages)) {
    throw new Error('Expected messages to be an array');
  }
  if (!messages.every(isMessage)) {
    throw new Error('Expected messages to be objects with role/content');
  }
  return messages;
};

const isTextContent = (value: unknown): value is OpenRouterContent =>
  isRecord(value) && value.type === 'text' && typeof value.text === 'string';

const isImageContent = (value: unknown): value is OpenRouterContent =>
  isRecord(value) &&
  value.type === 'image_url' &&
  isRecord(value.image_url) &&
  typeof value.image_url.url === 'string';

const getContentItems = (content: unknown): OpenRouterContent[] => {
  if (!Array.isArray(content)) {
    throw new Error('Expected message content to be an array');
  }
  return content.filter((item): item is OpenRouterContent => isTextContent(item) || isImageContent(item));
};

const getHeaderRecord = (headers: HeadersInit | undefined): Record<string, string> => {
  if (!headers) {
    return {};
  }
  if (headers instanceof Headers) {
    const record: Record<string, string> = {};
    headers.forEach((value, key) => {
      record[key] = value;
    });
    return record;
  }
  if (Array.isArray(headers)) {
    const record: Record<string, string> = {};
    for (const [key, value] of headers) {
      record[key] = value;
    }
    return record;
  }
  return headers;
};

describe('OpenRouterVisionClient', () => {
  const validConfig = {
    apiKey: 'test-api-key',
    modelId: 'qwen3-vl-30b',
  };

  const validRequest = {
    screenshot: Buffer.from([0x89, 0x50, 0x4e, 0x47]), // PNG magic bytes
    goal: 'Click the login button',
    currentUrl: 'https://example.com',
    conversationHistory: [],
    elementLabels: [
      {
        id: 1,
        selector: '#login-btn',
        tagName: 'button',
        bounds: { x: 100, y: 200, width: 80, height: 30 },
        text: 'Login',
      },
      {
        id: 2,
        selector: '#signup-btn',
        tagName: 'button',
        bounds: { x: 200, y: 200, width: 80, height: 30 },
        text: 'Sign Up',
      },
    ],
  };

  const successResponse = {
    id: 'chatcmpl-123',
    model: 'qwen/qwen3-vl-30b-a3b-instruct',
    choices: [
      {
        index: 0,
        message: {
          role: 'assistant',
          content: `I can see two buttons on the page: "Login" [1] and "Sign Up" [2].

Since the goal is to click the login button, I should click on element [1].

ACTION: click(1)`,
        },
        finish_reason: 'stop',
      },
    ],
    usage: {
      prompt_tokens: 1500,
      completion_tokens: 50,
      total_tokens: 1550,
    },
  };

  beforeEach(() => {
    mockFetch.mockReset();
  });

  describe('constructor', () => {
    it('creates client with valid config', () => {
      const client = new OpenRouterVisionClient(validConfig);
      expect(client.getModelSpec().id).toBe('qwen3-vl-30b');
    });

    it('throws for non-OpenRouter model', () => {
      expect(() => {
        new OpenRouterVisionClient({
          apiKey: 'test',
          modelId: 'claude-sonnet-4', // Anthropic model
        });
      }).toThrow(VisionModelError);
    });

    it('throws for unknown model', () => {
      expect(() => {
        new OpenRouterVisionClient({
          apiKey: 'test',
          modelId: 'unknown-model',
        });
      }).toThrow();
    });
  });

  describe('analyze', () => {
    it('parses click action from response', async () => {
      mockFetch.mockResolvedValueOnce(createJsonResponse(successResponse));

      const client = new OpenRouterVisionClient(validConfig);
      const result = await client.analyze(validRequest);

      expect(result.action).toEqual({
        type: 'click',
        elementId: 1,
      });
      expect(result.reasoning).toContain('Login');
      expect(result.goalAchieved).toBe(false);
      expect(result.tokensUsed.totalTokens).toBe(1550);
    });

    it('parses type action from response', async () => {
      mockFetch.mockResolvedValueOnce(
        createJsonResponse({
          ...successResponse,
          choices: [
            {
              ...successResponse.choices[0],
              message: {
                role: 'assistant',
                content: 'Typing email into input field.\n\nACTION: type(3, "test@example.com")',
              },
            },
          ],
        })
      );

      const client = new OpenRouterVisionClient(validConfig);
      const result = await client.analyze(validRequest);

      expect(result.action).toEqual({
        type: 'type',
        elementId: 3,
        text: 'test@example.com',
      });
    });

    it('parses done action and sets goalAchieved', async () => {
      mockFetch.mockResolvedValueOnce(
        createJsonResponse({
          ...successResponse,
          choices: [
            {
              ...successResponse.choices[0],
              message: {
                role: 'assistant',
                content: 'Task completed successfully.\n\nACTION: done(true, "Logged in")',
              },
            },
          ],
        })
      );

      const client = new OpenRouterVisionClient(validConfig);
      const result = await client.analyze(validRequest);

      expect(result.action).toEqual({
        type: 'done',
        success: true,
        result: 'Logged in',
      });
      expect(result.goalAchieved).toBe(true);
    });

    it('parses JSON block response format', async () => {
      mockFetch.mockResolvedValueOnce(
        createJsonResponse({
          ...successResponse,
          choices: [
            {
              ...successResponse.choices[0],
              message: {
                role: 'assistant',
                content: `Scrolling down to find more content.

\`\`\`json
{"type": "scroll", "direction": "down"}
\`\`\``,
              },
            },
          ],
        })
      );

      const client = new OpenRouterVisionClient(validConfig);
      const result = await client.analyze(validRequest);

      expect(result.action).toEqual({
        type: 'scroll',
        direction: 'down',
      });
    });

    it('includes element labels in request', async () => {
      mockFetch.mockResolvedValueOnce(createJsonResponse(successResponse));

      const client = new OpenRouterVisionClient(validConfig);
      await client.analyze(validRequest);

      expect(mockFetch).toHaveBeenCalledTimes(1);
      const body = getRequestBody();
      const messages = getMessages(body);

      const userMessage = messages.find((message) => message.role === 'user');
      if (!userMessage) {
        throw new Error('Expected user message to be present');
      }

      const contentItems = getContentItems(userMessage.content);
      const textContent = contentItems.find(isTextContent);
      if (!textContent || textContent.type !== 'text') {
        throw new Error('Expected text content in user message');
      }

      expect(textContent.text).toContain('[1]');
      expect(textContent.text).toContain('Login');
    });

    it('includes screenshot as base64 image', async () => {
      mockFetch.mockResolvedValueOnce(createJsonResponse(successResponse));

      const client = new OpenRouterVisionClient(validConfig);
      await client.analyze(validRequest);

      const body = getRequestBody();
      const messages = getMessages(body);
      const userMessage = messages.find((message) => message.role === 'user');
      if (!userMessage) {
        throw new Error('Expected user message to be present');
      }

      const contentItems = getContentItems(userMessage.content);
      const imageContent = contentItems.find(isImageContent);
      if (!imageContent || imageContent.type !== 'image_url') {
        throw new Error('Expected image content in user message');
      }

      expect(imageContent.image_url.url).toMatch(/^data:image\/png;base64,/);
    });

    it('sends correct headers', async () => {
      mockFetch.mockResolvedValueOnce(createJsonResponse(successResponse));

      const client = new OpenRouterVisionClient(validConfig);
      await client.analyze(validRequest);

      const options = getRequestOptions();
      const headers = getHeaderRecord(options.headers);
      expect(headers['Authorization']).toBe('Bearer test-api-key');
      expect(headers['Content-Type']).toBe('application/json');
      expect(headers['HTTP-Referer']).toBe('https://vrooli.com');
    });

    it('includes conversation history in messages', async () => {
      mockFetch.mockResolvedValueOnce(createJsonResponse(successResponse));

      const client = new OpenRouterVisionClient(validConfig);
      await client.analyze({
        ...validRequest,
        conversationHistory: [
          { role: 'user', content: 'First prompt' },
          { role: 'assistant', content: 'First response\n\nACTION: scroll(down)' },
        ],
      });

      const body = getRequestBody();
      const messages = getMessages(body);

      // Should have: system + history user + history assistant + current user
      expect(messages.length).toBe(4);
      const [systemMessage, historyUser, historyAssistant] = messages;
      if (!systemMessage || !historyUser || !historyAssistant) {
        throw new Error('Expected system and history messages');
      }
      expect(systemMessage.role).toBe('system');
      expect(historyUser.role).toBe('user');
      expect(historyUser.content).toBe('First prompt');
      expect(historyAssistant.role).toBe('assistant');
    });
  });

  describe('error handling', () => {
    it('throws INVALID_API_KEY on 401', async () => {
      mockFetch.mockResolvedValueOnce(
        createTextResponse(JSON.stringify({ error: { message: 'Invalid API key' } }), 401)
      );

      const client = new OpenRouterVisionClient(validConfig);

      try {
        await client.analyze(validRequest);
        fail('Expected VisionModelError to be thrown');
      } catch (error) {
        expect(error).toBeInstanceOf(VisionModelError);
        expect((error as VisionModelError).code).toBe('INVALID_API_KEY');
        expect((error as VisionModelError).retryable).toBe(false);
      }
    });

    it('throws RATE_LIMITED on 429', async () => {
      mockFetch.mockResolvedValueOnce(createTextResponse('Rate limit exceeded', 429));

      const client = new OpenRouterVisionClient({
        ...validConfig,
        maxRetries: 0, // No retries to test immediate error
      });

      try {
        await client.analyze(validRequest);
        fail('Expected VisionModelError to be thrown');
      } catch (error) {
        expect(error).toBeInstanceOf(VisionModelError);
        expect((error as VisionModelError).code).toBe('RATE_LIMITED');
        expect((error as VisionModelError).retryable).toBe(true);
      }
    });

    it('throws QUOTA_EXCEEDED on 402', async () => {
      mockFetch.mockResolvedValueOnce(createTextResponse('Insufficient credits', 402));

      const client = new OpenRouterVisionClient(validConfig);

      await expect(client.analyze(validRequest)).rejects.toMatchObject({
        code: 'QUOTA_EXCEEDED',
        retryable: false,
      });
    });

    it('retries on 500 errors', async () => {
      mockFetch
        .mockResolvedValueOnce(createTextResponse('Internal server error', 500))
        .mockResolvedValueOnce(createJsonResponse(successResponse));

      const client = new OpenRouterVisionClient({
        ...validConfig,
        maxRetries: 1,
      });

      const result = await client.analyze(validRequest);
      expect(result.action.type).toBe('click');
      expect(mockFetch).toHaveBeenCalledTimes(2);
    });

    it('throws PARSE_ERROR when action cannot be parsed', async () => {
      mockFetch.mockResolvedValueOnce(
        createJsonResponse({
          ...successResponse,
          choices: [
            {
              ...successResponse.choices[0],
              message: {
                role: 'assistant',
                content: 'I am confused and do not know what to do.',
              },
            },
          ],
        })
      );

      const client = new OpenRouterVisionClient(validConfig);

      await expect(client.analyze(validRequest)).rejects.toMatchObject({
        code: 'PARSE_ERROR',
      });
    });

    it('throws when no choices returned', async () => {
      mockFetch.mockResolvedValueOnce(
        createJsonResponse({
          ...successResponse,
          choices: [],
        })
      );

      const client = new OpenRouterVisionClient(validConfig);

      await expect(client.analyze(validRequest)).rejects.toMatchObject({
        code: 'PARSE_ERROR',
      });
    });
  });

  describe('token estimation', () => {
    it('estimates tokens when usage not provided', async () => {
      mockFetch.mockResolvedValueOnce(
        createJsonResponse({
          ...successResponse,
          usage: undefined,
        })
      );

      const client = new OpenRouterVisionClient(validConfig);
      const result = await client.analyze(validRequest);

      expect(result.tokensUsed.promptTokens).toBeGreaterThan(0);
      expect(result.tokensUsed.completionTokens).toBeGreaterThan(0);
      expect(result.tokensUsed.totalTokens).toBe(
        result.tokensUsed.promptTokens + result.tokensUsed.completionTokens
      );
    });
  });

  describe('confidence estimation', () => {
    it('gives high confidence for click with elementId', async () => {
      mockFetch.mockResolvedValueOnce(createJsonResponse(successResponse));

      const client = new OpenRouterVisionClient(validConfig);
      const result = await client.analyze(validRequest);

      expect(result.confidence).toBeGreaterThan(0.85);
    });

    it('gives lower confidence for scroll actions', async () => {
      mockFetch.mockResolvedValueOnce(
        createJsonResponse({
          ...successResponse,
          choices: [
            {
              ...successResponse.choices[0],
              message: {
                role: 'assistant',
                content: 'Need to scroll.\n\nACTION: scroll(down)',
              },
            },
          ],
        })
      );

      const client = new OpenRouterVisionClient(validConfig);
      const result = await client.analyze(validRequest);

      expect(result.confidence).toBeLessThan(0.7);
    });

    it('gives high confidence for successful done action', async () => {
      mockFetch.mockResolvedValueOnce(
        createJsonResponse({
          ...successResponse,
          choices: [
            {
              ...successResponse.choices[0],
              message: {
                role: 'assistant',
                // Reasoning must be > 50 chars to avoid confidence penalty
                content: 'I have successfully completed the task. The login form was submitted and I can now see the dashboard with the user profile. Everything appears to be working correctly.\n\nACTION: done(true, "Login completed successfully")',
              },
            },
          ],
        })
      );

      const client = new OpenRouterVisionClient(validConfig);
      const result = await client.analyze(validRequest);

      expect(result.confidence).toBeGreaterThan(0.9);
    });
  });

  describe('createOpenRouterClient factory', () => {
    it('creates client via factory function', () => {
      const client = createOpenRouterClient(validConfig);
      expect(client).toBeInstanceOf(OpenRouterVisionClient);
      expect(client.getModelSpec().id).toBe('qwen3-vl-30b');
    });
  });
});
