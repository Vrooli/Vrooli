import { describe, expect, it, vi } from 'vitest';
import { ApiError } from './common';
import { withReauthentication } from './reauthentication';

describe('withReauthentication', () => {
  it('reauthenticates and retries once for the server reason', async () => {
    const action = vi.fn()
      .mockRejectedValueOnce(new ApiError('recent auth required', 'forbidden', 403, 'reauthentication_required', false, { reason: 'reauthentication_required' }))
      .mockResolvedValueOnce('ok');
    const reauthenticate = vi.fn().mockResolvedValue(undefined);
    await expect(withReauthentication(action, reauthenticate)).resolves.toBe('ok');
    expect(reauthenticate).toHaveBeenCalledOnce();
    expect(action).toHaveBeenCalledTimes(2);
  });

  it('does not retry when the dialog is cancelled or unavailable', async () => {
    const error = new ApiError('recent auth required', 'forbidden', 403, 'reauthentication_required', false, { reason: 'reauthentication_required' });
    const action = vi.fn().mockRejectedValue(error);
    await expect(withReauthentication(action)).rejects.toBe(error);
    expect(action).toHaveBeenCalledOnce();
  });
});
