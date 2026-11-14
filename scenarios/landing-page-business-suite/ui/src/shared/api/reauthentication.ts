import { ConnectError, Code } from '@connectrpc/connect';
import { ApiError } from './common';

export type Reauthenticate = () => Promise<void>;

function requiresReauthentication(error: unknown): boolean {
	if (error instanceof ApiError) return error.reason === 'reauthentication_required' || error.reason === 'admin_reauthentication_required';
	if (error instanceof ConnectError) {
		return error.metadata.get('x-lpbs-auth-reason') === 'reauthentication_required' || error.metadata.get('x-lpbs-auth-reason') === 'admin_reauthentication_required' || error.code === Code.FailedPrecondition;
  }
  return false;
}

/** Retry one sensitive operation after exactly one successful step-up. */
export async function withReauthentication<T>(action: () => Promise<T>, reauthenticate?: Reauthenticate): Promise<T> {
  try {
    return await action();
  } catch (error) {
    if (!requiresReauthentication(error) || !reauthenticate) throw error;
    await reauthenticate();
    return action();
  }
}
