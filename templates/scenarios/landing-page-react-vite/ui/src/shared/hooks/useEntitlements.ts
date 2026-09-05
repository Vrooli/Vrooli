import { useCallback, useEffect, useState } from 'react';
import { getEntitlements, type GetEntitlementsResponse } from '../api';

const STORAGE_KEY = 'landing_entitlement_email';

function readStoredEmail(): string {
  if (typeof window === 'undefined') {
    return '';
  }
  try {
    return window.localStorage.getItem(STORAGE_KEY) ?? '';
  } catch {
    return '';
  }
}

function persistEmail(email: string) {
  if (typeof window === 'undefined') {
    return;
  }
  try {
    if (email === '') {
      window.localStorage.removeItem(STORAGE_KEY);
    } else {
      window.localStorage.setItem(STORAGE_KEY, email);
    }
  } catch {
    // Ignore localStorage failures
  }
}

export function useEntitlements() {
  const [email, setEmailState] = useState<string>(() => readStoredEmail());
  const [entitlements, setEntitlements] = useState<GetEntitlementsResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const setEmail = useCallback((value: string) => {
    setEmailState(value);
    persistEmail(value);
  }, []);

  const refresh = useCallback(async () => {
    const trimmed = email.trim();

    if (!trimmed) {
      setEntitlements(null);
      setError(null);
      setLoading(false);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      // Entitlements are resolved from the session identity server-side; the
      // stored email gates the client-side fetch but is no longer passed.
      const payload = await getEntitlements();
      setEntitlements(payload);
    } catch (err) {
      setEntitlements(null);
      setError(err instanceof Error ? err.message : 'Failed to load entitlements');
    } finally {
      setLoading(false);
    }
  }, [email]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  return { email, setEmail, entitlements, loading, error, refresh };
}
