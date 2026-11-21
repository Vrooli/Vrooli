import * as React from "react";
import { useParams } from "react-router-dom";

/**
 * useScenarioPath — reads `:encodedPath` from the URL and decodes it back to
 * the scenario filesystem path the API expects. Encodes via
 * `encodeURIComponent` so paths can contain `/`, spaces, etc.
 *
 * Returns `null` when the param is absent (the route is not under
 * `/targets/:encodedPath/…`) or when decoding throws.
 */
export function useScenarioPath(): string | null {
  const params = useParams<{ encodedPath?: string }>();
  return React.useMemo(() => {
    const raw = params.encodedPath;
    if (raw === undefined) return null;
    try {
      const decoded = decodeURIComponent(raw);
      return decoded.length === 0 ? null : decoded;
    } catch {
      return null;
    }
  }, [params.encodedPath]);
}

/** Encode a scenario filesystem path for embedding in the `:encodedPath` URL slot. */
export function encodeScenarioPath(path: string): string {
  return encodeURIComponent(path);
}
