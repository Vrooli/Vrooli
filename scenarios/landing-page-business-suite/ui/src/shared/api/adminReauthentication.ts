import { ApiError } from './common';

type ReauthRequester = () => Promise<void>;
let requester: ReauthRequester | undefined;

export function registerAdminReauthRequester(next?: ReauthRequester): () => void {
  requester = next;
  return () => { if (requester === next) requester = undefined; };
}

function needsReauth(error: unknown): boolean {
  if (error instanceof ApiError) return error.reason === 'admin_reauthentication_required';
  if (!error || typeof error !== 'object' || !('metadata' in error)) return false;
  const metadata = (error as { metadata?: { get?: (key: string) => string | undefined } }).metadata;
  return metadata?.get?.('x-lpbs-auth-reason') === 'admin_reauthentication_required';
}

/** Retry an administrator operation once after the global reauthentication dialog succeeds. */
export async function withAdminReauthentication<T>(action: () => Promise<T>): Promise<T> {
  try { return await action(); } catch (error) {
    if (!needsReauth(error) || !requester) throw error;
    await requester();
    return action();
  }
}
