/**
 * Claude Computer Use Vision Client Tests
 */

import {
  ClaudeComputerUseClient,
  createClaudeComputerUseClient,
} from '../../../../src/ai/vision-client/claude-computer-use';
import { VisionModelError } from '../../../../src/ai/vision-client/types';
import {
  fetchJsonResponse,
  fetchTextResponse,
  getFetchRequestBodyJson,
  installFetchMock,
} from '../../../helpers';

const mockFetch = installFetchMock();

describe('ClaudeComputerUseClient', () => {
  const validConfig = {
    apiKey: 'test-api-key',
    modelId: 'claude-sonnet-4',
  };

  const validRequest = {
    screenshot: Buffer.from([0x89, 0x50, 0x4e, 0x47]),
    goal: 'Click the login button',
    currentUrl: 'https://example.com',
    conversationHistory: [
      {
        role: 'user' as const,
        content: 'Here is the page',
        screenshot: Buffer.from([0xff, 0xd8, 0x00]),
      },
      {
        role: 'assistant' as const,
        content: 'Looking now',
      },
    ],
    elementLabels: [
      {
        id: 1,
        selector: '#login-btn',
        tagName: 'button',
        bounds: { x: 100, y: 200, width: 80, height: 30 },
        text: 'Login',
      },
    ],
  };

  beforeEach(() => {
    mockFetch.mockReset();
  });

  describe('constructor', () => {
    it('creates client with valid config', () => {
      const client = new ClaudeComputerUseClient(validConfig);
      expect(client.getModelSpec().id).toBe('claude-sonnet-4');
    });

    it('throws for non-Anthropic model', () => {
      expect(() => {
        new ClaudeComputerUseClient({
          apiKey: 'test',
          modelId: 'qwen3-vl-30b',
        });
      }).toThrow(VisionModelError);
    });
  });

  describe('analyze', () => {
    it('parses tool_use actions into browser actions', async () => {
      mockFetch.mockResolvedValueOnce(
        fetchJsonResponse({
          id: 'msg-1',
          type: 'message',
          role: 'assistant',
          content: [
            { type: 'text', text: 'Click the login button.' },
            {
              type: 'tool_use',
              id: 'tool-1',
              name: 'computer',
              input: { action: 'left_click', coordinate: [10, 20] },
            },
          ],
          model: 'claude-sonnet-4',
          stop_reason: 'tool_use',
          usage: { input_tokens: 10, output_tokens: 5 },
        })
      );

      const client = new ClaudeComputerUseClient(validConfig);
      const result = await client.analyze(validRequest);

      expect(result.action).toEqual({
        type: 'click',
        coordinates: { x: 10, y: 20 },
        variant: 'left',
      });
      expect(result.reasoning).toContain('Click the login');
      expect(result.goalAchieved).toBe(false);
      expect(result.tokensUsed.totalTokens).toBe(15);

      const body = getFetchRequestBodyJson(mockFetch);
      expect(body.model).toBe('anthropic/claude-sonnet-4-20250514');
      expect(body.tools).toEqual([
        expect.objectContaining({
          type: 'computer_20251124',
          name: 'computer',
          display_width_px: 1280,
          display_height_px: 800,
        }),
      ]);
    });

    it('marks goal achieved when reasoning indicates completion', async () => {
      mockFetch.mockResolvedValueOnce(
        fetchJsonResponse({
          id: 'msg-2',
          type: 'message',
          role: 'assistant',
          content: [{ type: 'text', text: 'Goal achieved. Login succeeded.' }],
          model: 'claude-sonnet-4',
          stop_reason: 'end_turn',
          usage: { input_tokens: 5, output_tokens: 5 },
        })
      );

      const client = createClaudeComputerUseClient(validConfig);
      const result = await client.analyze(validRequest);

      expect(result.action).toEqual({
        type: 'done',
        success: true,
        result: expect.stringContaining('Goal achieved'),
      });
      expect(result.goalAchieved).toBe(true);
      expect(result.confidence).toBeCloseTo(0.855, 5);
    });

    it('defaults to wait when no tool use or completion signal is present', async () => {
      mockFetch.mockResolvedValueOnce(
        fetchJsonResponse({
          id: 'msg-3',
          type: 'message',
          role: 'assistant',
          content: [{ type: 'text', text: 'I need more info.' }],
          model: 'claude-sonnet-4',
          stop_reason: 'end_turn',
          usage: { input_tokens: 5, output_tokens: 5 },
        })
      );

      const client = new ClaudeComputerUseClient(validConfig);
      const result = await client.analyze(validRequest);

      expect(result.action).toEqual({ type: 'wait', ms: 1000 });
      expect(result.goalAchieved).toBe(false);
    });

    it('includes proper image media types in the request', async () => {
      mockFetch.mockResolvedValueOnce(
        fetchJsonResponse({
          id: 'msg-4',
          type: 'message',
          role: 'assistant',
          content: [
            {
              type: 'tool_use',
              id: 'tool-2',
              name: 'computer',
              input: { action: 'wait', duration: 1 },
            },
          ],
          model: 'claude-sonnet-4',
          stop_reason: 'tool_use',
          usage: { input_tokens: 1, output_tokens: 1 },
        })
      );

      const client = new ClaudeComputerUseClient(validConfig);
      await client.analyze(validRequest);

      const body = getFetchRequestBodyJson(mockFetch);
      const messages = body.messages as Array<{ content: Array<{ type: string; source?: { media_type?: string } }> }>;
      const imageTypes = messages
        .flatMap((message) => message.content)
        .filter((content) => content.type === 'image')
        .map((content) => content.source?.media_type)
        .filter((type): type is string => typeof type === 'string');

      expect(imageTypes).toContain('image/png');
      expect(imageTypes).toContain('image/jpeg');
    });

    it('throws on API error responses', async () => {
      mockFetch.mockResolvedValueOnce(
        fetchTextResponse(JSON.stringify({ error: { message: 'bad key' } }), 401)
      );

      const client = new ClaudeComputerUseClient(validConfig);

      await expect(client.analyze(validRequest)).rejects.toMatchObject({
        code: 'INVALID_API_KEY',
        retryable: false,
      });
    });

    it('throws on context-too-long errors', async () => {
      mockFetch.mockResolvedValueOnce(
        fetchTextResponse(JSON.stringify({ error: { message: 'context length exceeded' } }), 400)
      );

      const client = new ClaudeComputerUseClient(validConfig);

      await expect(client.analyze(validRequest)).rejects.toMatchObject({
        code: 'CONTEXT_TOO_LONG',
        retryable: false,
      });
    });

    it('throws when API returns error payload', async () => {
      mockFetch.mockResolvedValueOnce(
        fetchJsonResponse({
          id: 'msg-5',
          type: 'message',
          role: 'assistant',
          content: [],
          model: 'claude-sonnet-4',
          stop_reason: 'end_turn',
          usage: { input_tokens: 1, output_tokens: 1 },
          error: { type: 'rate_limit', message: 'slow down' },
        })
      );

      const client = new ClaudeComputerUseClient(validConfig);

      await expect(client.analyze(validRequest)).rejects.toMatchObject({
        code: 'UNKNOWN',
        retryable: false,
      });
    });
  });
});
