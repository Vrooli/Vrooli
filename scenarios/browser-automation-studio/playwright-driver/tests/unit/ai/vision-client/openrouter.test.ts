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

// Mock fetch globally
const mockFetch = jest.fn();
global.fetch = mockFetch;

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
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => successResponse,
      });

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
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
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
        }),
      });

      const client = new OpenRouterVisionClient(validConfig);
      const result = await client.analyze(validRequest);

      expect(result.action).toEqual({
        type: 'type',
        elementId: 3,
        text: 'test@example.com',
      });
    });

    it('parses done action and sets goalAchieved', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
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
        }),
      });

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
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
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
        }),
      });

      const client = new OpenRouterVisionClient(validConfig);
      const result = await client.analyze(validRequest);

      expect(result.action).toEqual({
        type: 'scroll',
        direction: 'down',
      });
    });

    it('includes element labels in request', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => successResponse,
      });

      const client = new OpenRouterVisionClient(validConfig);
      await client.analyze(validRequest);

      expect(mockFetch).toHaveBeenCalledTimes(1);
      const [, options] = mockFetch.mock.calls[0];
      const body = JSON.parse(options.body);

      // Check that messages include element labels
      const userMessage = body.messages.find((m: { role: string }) => m.role === 'user');
      expect(userMessage).toBeDefined();

      // Find the text content that should contain element labels
      const textContent = userMessage.content.find((c: { type: string }) => c.type === 'text');
      expect(textContent.text).toContain('[1]');
      expect(textContent.text).toContain('Login');
    });

    it('includes screenshot as base64 image', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => successResponse,
      });

      const client = new OpenRouterVisionClient(validConfig);
      await client.analyze(validRequest);

      const [, options] = mockFetch.mock.calls[0];
      const body = JSON.parse(options.body);
      const userMessage = body.messages.find((m: { role: string }) => m.role === 'user');
      const imageContent = userMessage.content.find((c: { type: string }) => c.type === 'image_url');

      expect(imageContent.image_url.url).toMatch(/^data:image\/png;base64,/);
    });

    it('sends correct headers', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => successResponse,
      });

      const client = new OpenRouterVisionClient(validConfig);
      await client.analyze(validRequest);

      const [, options] = mockFetch.mock.calls[0];
      expect(options.headers['Authorization']).toBe('Bearer test-api-key');
      expect(options.headers['Content-Type']).toBe('application/json');
      expect(options.headers['HTTP-Referer']).toBe('https://vrooli.com');
    });

    it('includes conversation history in messages', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => successResponse,
      });

      const client = new OpenRouterVisionClient(validConfig);
      await client.analyze({
        ...validRequest,
        conversationHistory: [
          { role: 'user', content: 'First prompt' },
          { role: 'assistant', content: 'First response\n\nACTION: scroll(down)' },
        ],
      });

      const [, options] = mockFetch.mock.calls[0];
      const body = JSON.parse(options.body);

      // Should have: system + history user + history assistant + current user
      expect(body.messages.length).toBe(4);
      expect(body.messages[0].role).toBe('system');
      expect(body.messages[1].role).toBe('user');
      expect(body.messages[1].content).toBe('First prompt');
      expect(body.messages[2].role).toBe('assistant');
    });
  });

  describe('error handling', () => {
    it('throws INVALID_API_KEY on 401', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 401,
        text: async () => JSON.stringify({ error: { message: 'Invalid API key' } }),
      });

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
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 429,
        text: async () => 'Rate limit exceeded',
      });

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
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 402,
        text: async () => 'Insufficient credits',
      });

      const client = new OpenRouterVisionClient(validConfig);

      await expect(client.analyze(validRequest)).rejects.toMatchObject({
        code: 'QUOTA_EXCEEDED',
        retryable: false,
      });
    });

    it('retries on 500 errors', async () => {
      mockFetch
        .mockResolvedValueOnce({
          ok: false,
          status: 500,
          text: async () => 'Internal server error',
        })
        .mockResolvedValueOnce({
          ok: true,
          json: async () => successResponse,
        });

      const client = new OpenRouterVisionClient({
        ...validConfig,
        maxRetries: 1,
      });

      const result = await client.analyze(validRequest);
      expect(result.action.type).toBe('click');
      expect(mockFetch).toHaveBeenCalledTimes(2);
    });

    it('throws PARSE_ERROR when action cannot be parsed', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
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
        }),
      });

      const client = new OpenRouterVisionClient(validConfig);

      await expect(client.analyze(validRequest)).rejects.toMatchObject({
        code: 'PARSE_ERROR',
      });
    });

    it('throws when no choices returned', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          ...successResponse,
          choices: [],
        }),
      });

      const client = new OpenRouterVisionClient(validConfig);

      await expect(client.analyze(validRequest)).rejects.toMatchObject({
        code: 'PARSE_ERROR',
      });
    });
  });

  describe('token estimation', () => {
    it('estimates tokens when usage not provided', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          ...successResponse,
          usage: undefined,
        }),
      });

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
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => successResponse,
      });

      const client = new OpenRouterVisionClient(validConfig);
      const result = await client.analyze(validRequest);

      expect(result.confidence).toBeGreaterThan(0.85);
    });

    it('gives lower confidence for scroll actions', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
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
        }),
      });

      const client = new OpenRouterVisionClient(validConfig);
      const result = await client.analyze(validRequest);

      expect(result.confidence).toBeLessThan(0.7);
    });

    it('gives high confidence for successful done action', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({
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
        }),
      });

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
