export function isPasskeySupported(): boolean {
  return typeof window !== 'undefined' && typeof navigator !== 'undefined' && typeof navigator.credentials?.get === 'function' && typeof navigator.credentials?.create === 'function';
}

export function isConditionalAvailable(): Promise<boolean> {
  const credential = (globalThis as typeof globalThis & { PublicKeyCredential?: { isConditionalMediationAvailable?: () => Promise<boolean> } }).PublicKeyCredential;
  return credential?.isConditionalMediationAvailable?.() ?? Promise.resolve(false);
}

export function base64urlToBytes(value: string): Uint8Array {
  const normalized = value.replace(/-/g, '+').replace(/_/g, '/') + '='.repeat((4 - value.length % 4) % 4);
  return Uint8Array.from(atob(normalized), (char) => char.charCodeAt(0));
}

export function bytesToBase64url(value: ArrayBuffer | Uint8Array): string {
  const bytes = value instanceof Uint8Array ? value : new Uint8Array(value);
  let binary = '';
  bytes.forEach((byte) => { binary += String.fromCharCode(byte); });
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

export function toCreationOptions(options: Record<string, unknown>): Record<string, unknown> {
  const publicKey = (options.publicKey && typeof options.publicKey === 'object' ? options.publicKey : options) as Record<string, unknown>;
  const user = publicKey.user as Record<string, unknown> | undefined;
  if (user?.id && typeof user.id === 'string') user.id = base64urlToBytes(user.id);
  const challenge = publicKey.challenge;
  if (typeof challenge === 'string') publicKey.challenge = base64urlToBytes(challenge);
  return publicKey;
}

export function toRequestOptions(options: Record<string, unknown>): Record<string, unknown> {
  const publicKey = (options.publicKey && typeof options.publicKey === 'object' ? options.publicKey : options) as Record<string, unknown>;
  if (typeof publicKey.challenge === 'string') publicKey.challenge = base64urlToBytes(publicKey.challenge);
  const allowCredentials = publicKey.allowCredentials;
  if (Array.isArray(allowCredentials)) {
    publicKey.allowCredentials = allowCredentials.map((credential) => {
      if (!credential || typeof credential !== 'object') return credential;
      const entry = credential as Record<string, unknown>;
      if (typeof entry.id === 'string') entry.id = base64urlToBytes(entry.id);
      return entry;
    });
  }
  return publicKey;
}

export function credentialToJSON(credential: PublicKeyCredential): Record<string, unknown> {
  const serializable = credential as PublicKeyCredential & { toJSON?: () => Record<string, unknown> };
  if (serializable.toJSON) return serializable.toJSON();
  return { id: credential.id, type: credential.type, rawId: bytesToBase64url(credential.rawId), response: credential.response };
}
