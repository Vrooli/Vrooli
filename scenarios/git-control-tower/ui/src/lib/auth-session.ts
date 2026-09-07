const ACCESS_TOKEN_COOKIE = "gct_access_token";

/**
 * Store the short-lived authenticator access token where GCT's relying-party
 * middleware already looks for it. The refresh token is deliberately not
 * stored: an operator can sign in again after the access token expires.
 */
export function saveGCTAccessToken(accessToken: string): void {
  const token = accessToken.trim();
  if (!token) throw new Error("Cannot store an empty authenticator access token");
  document.cookie = `${ACCESS_TOKEN_COOKIE}=${encodeURIComponent(token)}; Path=/; SameSite=Lax`;
}
