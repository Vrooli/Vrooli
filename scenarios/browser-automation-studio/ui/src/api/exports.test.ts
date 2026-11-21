import { afterEach, describe, expect, it, vi } from 'vitest';

import { exportsClient, generateExportCaption, _resetCaptionOperationIDs } from './exports';

describe('generateExportCaption operation identity', () => {
  afterEach(() => {
    vi.restoreAllMocks();
    _resetCaptionOperationIDs();
  });

  it('reuses the idempotency key after an uncertain error and rotates it after success', async () => {
    const call = vi.spyOn(exportsClient, 'generateExportCaption');
    call.mockRejectedValueOnce(new Error('outcome uncertain'));
    call.mockResolvedValueOnce({} as never);
    call.mockResolvedValueOnce({} as never);

    await expect(generateExportCaption('export-1')).rejects.toThrow('outcome uncertain');
    await generateExportCaption('export-1');
    await generateExportCaption('export-1');

    const keys = call.mock.calls.map(([, options]) => new Headers(options?.headers).get('Idempotency-Key'));
    expect(keys[0]).toMatch(/^[a-f0-9]{32}$/);
    expect(keys[1]).toBe(keys[0]);
    expect(keys[2]).toMatch(/^[a-f0-9]{32}$/);
    expect(keys[2]).not.toBe(keys[1]);
  });
});
