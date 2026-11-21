import { ConnectError, Code } from '@connectrpc/connect';
import { describe, expect, it, vi } from 'vitest';
import { registerAdminReauthRequester, withAdminReauthentication } from './adminReauthentication';

describe('withAdminReauthentication', () => {
  it('requests one proof and retries the operation once', async () => {
    const error = new ConnectError('step up', Code.PermissionDenied, new Headers([['x-lpbs-auth-reason', 'admin_reauthentication_required']]));
    const action = vi.fn().mockRejectedValueOnce(error).mockResolvedValueOnce('ok');
    const requester = vi.fn().mockResolvedValue(undefined);
    const cleanup = registerAdminReauthRequester(requester);
    await expect(withAdminReauthentication(action)).resolves.toBe('ok');
    expect(requester).toHaveBeenCalledOnce(); expect(action).toHaveBeenCalledTimes(2);
    cleanup();
  });

  it('does not retry when the operator cancels', async () => {
    const error = new ConnectError('step up', Code.PermissionDenied, new Headers([['x-lpbs-auth-reason', 'admin_reauthentication_required']]));
    const action = vi.fn().mockRejectedValue(error); const requester = vi.fn().mockRejectedValue(new Error('cancelled'));
    const cleanup = registerAdminReauthRequester(requester);
    await expect(withAdminReauthentication(action)).rejects.toThrow('cancelled');
    expect(action).toHaveBeenCalledOnce(); cleanup();
  });
});
