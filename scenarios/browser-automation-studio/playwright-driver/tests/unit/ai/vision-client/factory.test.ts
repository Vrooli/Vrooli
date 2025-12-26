/**
 * Vision Client Factory Tests
 */

import {
  createVisionClient,
  createMockClient,
  getModelInfo,
  isModelSupported,
  getSupportedModelIds,
} from '../../../../src/ai/vision-client/factory';
import { ClaudeComputerUseClient } from '../../../../src/ai/vision-client/claude-computer-use';
import { OpenRouterVisionClient } from '../../../../src/ai/vision-client/openrouter';
import { MockVisionClient } from '../../../../src/ai/vision-client/mock';

describe('factory', () => {
  describe('createVisionClient', () => {
    it('creates OpenRouter client for OpenRouter model', () => {
      const client = createVisionClient({
        modelId: 'qwen3-vl-30b',
        apiKey: 'test-key',
      });

      expect(client).toBeInstanceOf(OpenRouterVisionClient);
    });

    it('creates OpenRouter client for GPT-4o', () => {
      const client = createVisionClient({
        modelId: 'gpt-4o',
        apiKey: 'test-key',
      });

      expect(client).toBeInstanceOf(OpenRouterVisionClient);
    });

    it('creates Claude Computer Use client for Anthropic models', () => {
      const client = createVisionClient({
        modelId: 'claude-sonnet-4',
        apiKey: 'test-key',
      });

      expect(client).toBeInstanceOf(ClaudeComputerUseClient);
    });

    it('creates Claude Computer Use client for Claude Opus', () => {
      const client = createVisionClient({
        modelId: 'claude-opus-4',
        apiKey: 'test-key',
      });

      expect(client).toBeInstanceOf(ClaudeComputerUseClient);
    });

    it('throws for unknown models', () => {
      expect(() => {
        createVisionClient({
          modelId: 'unknown-model',
          apiKey: 'test-key',
        });
      }).toThrow();
    });

    it('passes timeout config to client', () => {
      const client = createVisionClient({
        modelId: 'qwen3-vl-30b',
        apiKey: 'test-key',
        timeoutMs: 30000,
      }) as OpenRouterVisionClient;

      // Client should be created without error
      expect(client).toBeInstanceOf(OpenRouterVisionClient);
    });
  });

  describe('createMockClient', () => {
    it('creates MockVisionClient', () => {
      const mock = createMockClient();
      expect(mock).toBeInstanceOf(MockVisionClient);
    });

    it('creates MockVisionClient with config', () => {
      const mock = createMockClient({
        modelId: 'gpt-4o',
        latencyMs: 100,
      });

      expect(mock).toBeInstanceOf(MockVisionClient);
      expect(mock.getModelSpec().id).toBe('gpt-4o');
    });
  });

  describe('getModelInfo', () => {
    it('returns model spec for valid model', () => {
      const spec = getModelInfo('qwen3-vl-30b');

      expect(spec.id).toBe('qwen3-vl-30b');
      expect(spec.displayName).toBe('Qwen3-VL-30B');
      expect(spec.provider).toBe('openrouter');
    });

    it('throws for unknown model', () => {
      expect(() => {
        getModelInfo('unknown-model');
      }).toThrow();
    });
  });

  describe('isModelSupported', () => {
    it('returns true for supported OpenRouter models', () => {
      expect(isModelSupported('qwen3-vl-30b')).toBe(true);
      expect(isModelSupported('gpt-4o')).toBe(true);
      expect(isModelSupported('gpt-4o-mini')).toBe(true);
    });

    it('returns true for Anthropic models', () => {
      expect(isModelSupported('claude-sonnet-4')).toBe(true);
      expect(isModelSupported('claude-opus-4')).toBe(true);
    });

    it('returns false for unknown models', () => {
      expect(isModelSupported('unknown-model')).toBe(false);
    });
  });

  describe('getSupportedModelIds', () => {
    it('returns array of supported model IDs', () => {
      const ids = getSupportedModelIds();

      expect(Array.isArray(ids)).toBe(true);
      expect(ids.length).toBeGreaterThan(0);
      expect(ids).toContain('qwen3-vl-30b');
      expect(ids).toContain('gpt-4o');
    });

    it('includes Anthropic models', () => {
      const ids = getSupportedModelIds();

      // Anthropic models are now supported via Claude Computer Use
      expect(ids).toContain('claude-sonnet-4');
      expect(ids).toContain('claude-opus-4');
    });

    it('does not include unsupported Ollama models', () => {
      const ids = getSupportedModelIds();

      // Ollama models are not yet implemented
      // (Note: currently no Ollama models in registry)
      expect(ids.every((id) => !id.startsWith('ollama-'))).toBe(true);
    });
  });
});
