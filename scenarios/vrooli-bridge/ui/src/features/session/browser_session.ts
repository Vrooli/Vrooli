const STORAGE_KEY = "vrooli-bridge.operator-session";
const SESSION_TTL_SECONDS = 15 * 60;

export interface BrowserEnrollment {
  operatorId: string;
  identityProvider: string;
  mode: string;
  reference: string;
  enrolledAt: string;
  scopeCeiling: string[];
  privateKeyPkcs8: string;
}

export interface BrowserKeyMaterial {
  publicKey: Uint8Array;
  privateKeyPkcs8: string;
}

function storage(): Storage | null {
  try {
    return typeof window !== "undefined" ? window.localStorage : null;
  } catch {
    return null;
  }
}

function toBase64Url(value: ArrayBuffer | Uint8Array): string {
  const bytes = value instanceof Uint8Array ? value : new Uint8Array(value);
  let binary = "";
  for (const byte of bytes) binary += String.fromCharCode(byte);
  return btoa(binary).replace(/\+/gu, "-").replace(/\//gu, "_").replace(/=+$/u, "");
}

function fromBase64Url(value: string): Uint8Array {
  const padded = value.replace(/-/gu, "+").replace(/_/gu, "/") + "===";
  const binary = atob(padded.slice(0, padded.length - (padded.length % 4)));
  return Uint8Array.from(binary, (character) => character.charCodeAt(0));
}

function exactArrayBuffer(value: Uint8Array): ArrayBuffer {
  const copy = new Uint8Array(value.byteLength);
  copy.set(value);
  return copy.buffer;
}

function cryptoAlgorithm(): AlgorithmIdentifier {
  // Ed25519 is supported by current Chromium, Firefox, and WebKit releases;
  // keeping the algorithm literal behind this seam also makes unsupported
  // browsers produce a typed enrollment failure instead of a bearer fallback.
  return { name: "Ed25519" };
}

export async function generateBrowserKeyMaterial(): Promise<BrowserKeyMaterial> {
  const pair = (await globalThis.crypto.subtle.generateKey(cryptoAlgorithm(), true, ["sign", "verify"])) as CryptoKeyPair;
  const publicKey = new Uint8Array(await globalThis.crypto.subtle.exportKey("raw", pair.publicKey));
  const privateKeyPkcs8 = toBase64Url(await globalThis.crypto.subtle.exportKey("pkcs8", pair.privateKey));
  return { publicKey, privateKeyPkcs8 };
}

export function saveBrowserEnrollment(enrollment: BrowserEnrollment): void {
  const target = storage();
  if (!target) throw new Error("browser enrollment storage is unavailable");
  target.setItem(STORAGE_KEY, JSON.stringify(enrollment));
}

export function loadBrowserEnrollment(): BrowserEnrollment | null {
  const target = storage();
  if (!target) return null;
  const raw = target.getItem(STORAGE_KEY);
  if (!raw) return null;
  try {
    const value = JSON.parse(raw) as Partial<BrowserEnrollment>;
    if (!value.operatorId || !value.reference || !value.privateKeyPkcs8 || !value.identityProvider || !value.mode) {
      return null;
    }
    return {
      operatorId: value.operatorId,
      identityProvider: value.identityProvider,
      mode: value.mode,
      reference: value.reference,
      enrolledAt: value.enrolledAt ?? "",
      scopeCeiling: value.scopeCeiling ?? [],
      privateKeyPkcs8: value.privateKeyPkcs8,
    };
  } catch {
    return null;
  }
}

export async function mintBrowserSession(enrollment: BrowserEnrollment, now = Date.now()): Promise<string> {
  const issuedAt = Math.floor(now / 1000);
  const claims = {
    enrollment_reference: enrollment.reference,
    operator_id: enrollment.operatorId,
    scopes: enrollment.scopeCeiling,
    iat: issuedAt,
    exp: issuedAt + SESSION_TTL_SECONDS,
  };
  const encodedClaims = toBase64Url(new TextEncoder().encode(JSON.stringify(claims)));
  const privateKey = await globalThis.crypto.subtle.importKey(
    "pkcs8",
    exactArrayBuffer(fromBase64Url(enrollment.privateKeyPkcs8)),
    cryptoAlgorithm(),
    false,
    ["sign"],
  );
  const signature = await globalThis.crypto.subtle.sign(cryptoAlgorithm(), privateKey, new TextEncoder().encode(encodedClaims));
  return `OS1.${encodedClaims}.${toBase64Url(signature)}`;
}
