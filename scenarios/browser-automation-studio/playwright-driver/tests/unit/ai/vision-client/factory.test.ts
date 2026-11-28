/** Vision client factory tests. */

import {
  createVisionClient,
  getModelInfo,
  isModelSupported,
  getSupportedModelIds,
} from '../../../../src/ai/vision-client/factory';
import { AIGatewayVisionClient } from '../../../../src/ai/vision-client/gateway';

describe('factory', () => {
  it('creates the AI Gateway client for the local-first profile', () => {
    const client = createVisionClient({ modelId: 'local_first', gatewayUrl: 'http://gateway.test' });
    expect(client).toBeInstanceOf(AIGatewayVisionClient);
    expect(client.getModelSpec()).toMatchObject({ id: 'local_first', provider: 'ai-gateway', tier: 'local' });
  });

  it('creates the AI Gateway client for the hosted profile', () => {
    const client = createVisionClient({ modelId: 'remote_only', gatewayUrl: 'http://gateway.test' });
    expect(client).toBeInstanceOf(AIGatewayVisionClient);
    expect(client.getModelSpec()).toMatchObject({ id: 'remote_only', provider: 'ai-gateway', tier: 'remote' });
  });

  it('rejects concrete provider model identifiers', () => {
    expect(() => createVisionClient({ modelId: 'gpt-4o', gatewayUrl: 'http://gateway.test' })).toThrow();
    expect(() => getModelInfo('qwen3-vl-30b')).toThrow();
  });

  it('reports only provider-neutral profiles as supported', () => {
    expect(isModelSupported('local_first')).toBe(true);
    expect(isModelSupported('remote-only')).toBe(true);
    expect(isModelSupported('gpt-4o')).toBe(false);
    expect(isModelSupported('claude-sonnet-4')).toBe(false);
    expect(getSupportedModelIds()).toEqual(['local_first', 'remote_only']);
  });
});
