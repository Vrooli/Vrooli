import { AIGatewayVisionClient } from '../../../../src/ai/vision-client/gateway';
import { VisionModelError } from '../../../../src/ai/vision-client/types';
import { fetchJsonResponse, installFetchMock } from '../../../helpers';

const mockFetch = installFetchMock();

function png(width = 1280, height = 720): Buffer {
  const image = Buffer.alloc(24);
  image.writeUInt32BE(0x89504e47, 0);
  image.writeUInt32BE(width, 16);
  image.writeUInt32BE(height, 20);
  return image;
}

function jpeg(width = 1280, height = 720): Buffer {
  const image = Buffer.alloc(24);
  image[0] = 0xff;
  image[1] = 0xd8;
  image[2] = 0xff;
  image[3] = 0xc0;
  image.writeUInt16BE(17, 4);
  image[6] = 8;
  image.writeUInt16BE(height, 7);
  image.writeUInt16BE(width, 9);
  return image;
}

const request = {
  screenshot: png(),
  goal: 'Click the login button',
  currentUrl: 'https://example.com/login',
  operationId: 'a'.repeat(64),
  conversationHistory: [
    { role: 'user' as const, content: 'Here is the page', screenshot: png(640, 360) },
    { role: 'assistant' as const, content: 'The login button is visible.' },
  ],
};

describe('AIGatewayVisionClient', () => {
  beforeEach(() => mockFetch.mockReset());

  it('sends an ordered multimodal conversation to the gateway', async () => {
    mockFetch.mockResolvedValueOnce(
      fetchJsonResponse({
        validated: true,
        valueJson: JSON.stringify({
          action: { type: 'click', elementId: 1 },
          reasoning: 'Clicking login',
          goalAchieved: false,
          confidence: 0.9,
        }),
        usage: { inputTokens: '1200', outputTokens: '30' },
      })
    );

    const client = new AIGatewayVisionClient({
      gatewayUrl: 'http://ai-gateway.test',
      profile: 'remote_only',
      authorization: 'Bearer signed-session-token',
      aiSource: 'vrooli-credits',
      privacyConsentGrant: 'opaque-signed-receipt',
    });
    const result = await client.analyze(request);

    expect(result.action).toEqual({ type: 'click', elementId: 1 });
    expect(result.tokensUsed).toEqual({
      promptTokens: 1200,
      completionTokens: 30,
      totalTokens: 1230,
    });
    const body = mockFetch.mock.calls[0][1]?.body as string;
    const payload = JSON.parse(body);
    expect(payload.role).toBe('extract.structured');
    expect(payload.profile).toBe('PROFILE_REMOTE_ONLY');
    expect(payload.privacyClass).toBe('PRIVACY_CLASS_CONFIDENTIAL');
    expect(payload.privacyProvenance).toBe('browser_automation_studio.vision_navigation');
    expect(payload.rootOperationId).toBe(request.operationId);
    expect(payload.turns).toHaveLength(3);
    expect(payload.turns[0].attachments[0].mediaType).toBe('image/png');
    expect(payload.turns[2].attachments[0].inlineBytes).toBe(request.screenshot.toString('base64'));
    expect(payload).not.toHaveProperty('apiKey');
    expect(payload).not.toHaveProperty('model');
    expect(mockFetch.mock.calls[0][1]?.headers).toMatchObject({
      Authorization: 'Bearer signed-session-token',
    });
    expect(mockFetch.mock.calls[0][1]?.headers).toMatchObject({
      'X-Vrooli-AI-Source': 'vrooli-credits',
    });
    expect(mockFetch.mock.calls[0][1]?.headers).toMatchObject({
      'X-Vrooli-AI-Consent-Grant': 'opaque-signed-receipt',
    });
  });

  it('keeps the root operation identity visible when confidential remote routing is refused', async () => {
    mockFetch.mockResolvedValueOnce(
      fetchJsonResponse(
        { error: { message: 'confidential requests require a local provider' } },
        { status: 400 }
      )
    );
    const client = new AIGatewayVisionClient({
      gatewayUrl: 'http://ai-gateway.test',
      profile: 'remote_only',
    });
    await expect(client.analyze(request)).rejects.toThrow(
      `root_operation_id=${request.operationId}`
    );
  });

  it('preserves Gateway uncertain-outcome category and exact retry identity', async () => {
    mockFetch.mockResolvedValueOnce(
      fetchJsonResponse({
        error: {
          code: 'INFERENCE_ERROR_CODE_OUTCOME_UNCERTAIN',
          message: 'hosted outcome uncertain',
        },
      })
    );
    const client = new AIGatewayVisionClient({ gatewayUrl: 'http://ai-gateway.test' });
    try {
      await client.analyze(request);
      throw new Error('expected uncertain outcome');
    } catch (error) {
      expect(error).toBeInstanceOf(VisionModelError);
      expect(error).toMatchObject({ code: 'OUTCOME_UNCERTAIN', retryable: false });
      expect((error as Error).message).toContain(`root_operation_id=${request.operationId}`);
    }
  });

  it('accepts the JPEG screenshots produced by the browser capture path', async () => {
    mockFetch.mockResolvedValueOnce(
      fetchJsonResponse({
        validated: true,
        valueJson: JSON.stringify({
          action: { type: 'done', success: true },
          reasoning: 'The page is ready.',
          goalAchieved: true,
          confidence: 0.9,
        }),
      })
    );

    const client = new AIGatewayVisionClient({ gatewayUrl: 'http://ai-gateway.test' });
    await client.analyze({ ...request, screenshot: jpeg(800, 600) });

    const body = mockFetch.mock.calls[0][1]?.body as string;
    const payload = JSON.parse(body);
    expect(payload.turns.at(-1).attachments[0]).toMatchObject({
      mediaType: 'image/jpeg',
      width: 800,
      height: 600,
    });
  });

  it('maps the envelope completion signal onto a done action when needed', async () => {
    mockFetch.mockResolvedValueOnce(
      fetchJsonResponse({
        validated: true,
        valueJson: JSON.stringify({
          action: { type: 'done', result: 'The page is visible.' },
          reasoning: 'The application page is visible.',
          goalAchieved: true,
          confidence: 0.9,
        }),
      })
    );

    const client = new AIGatewayVisionClient({ gatewayUrl: 'http://ai-gateway.test' });
    const result = await client.analyze(request);

    expect(result.action).toEqual({
      type: 'done',
      result: 'The page is visible.',
      success: true,
    });
    expect(result.goalAchieved).toBe(true);
  });

  it('rejects screenshots whose dimensions cannot be determined', async () => {
    jest.useFakeTimers();
    try {
      const client = new AIGatewayVisionClient({ gatewayUrl: 'http://ai-gateway.test' });
      await expect(
        client.analyze({ ...request, screenshot: Buffer.from('not an image') })
      ).rejects.toMatchObject({ code: 'PARSE_ERROR' });
      expect(mockFetch).not.toHaveBeenCalled();
      expect(jest.getTimerCount()).toBe(0);
    } finally {
      jest.useRealTimers();
    }
  });

  it('requires a configured gateway URL', () => {
    expect(() => new AIGatewayVisionClient({ gatewayUrl: '' })).toThrow(VisionModelError);
  });
});
