export interface BackdropReference {
  id: string;
  uri?: string;
  url?: string;
  placement?: string;
  alt_text?: string;
  reserved_regions?: Array<{ x: number; y: number; width: number; height: number; kind?: string }>;
}

const configuredBackdropStudioURL = import.meta.env.VITE_BACKDROP_STUDIO_URL;
const backdropStudioURL = typeof configuredBackdropStudioURL === "string" ? configuredBackdropStudioURL.replace(/\/$/, "") : undefined;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

/** Resolve only metadata. Raster bytes stay behind the URI owned by Backdrop Studio. */
export async function resolveBackdropReference(
  reference: Pick<BackdropReference, "id">,
  fetcher: typeof fetch = fetch,
): Promise<BackdropReference | null> {
  if (!reference.id || !backdropStudioURL) return null;
  const response = await fetcher(`${backdropStudioURL}/vrooli.backdrop_studio.v1.release.ReleaseService/GetReference`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ id: reference.id }),
  });
  if (!response.ok) return null;
  const rawBody: unknown = await response.json();
  if (!isRecord(rawBody) || typeof rawBody.id !== "string" || typeof rawBody.uri !== "string") return null;
  const bodyID = rawBody.id;
  const bodyURI = rawBody.uri;
  const rawRegions = rawBody.reservedRegions;
  const regions = Array.isArray(rawRegions)
    ? rawRegions.filter((region): region is { x: number; y: number; width: number; height: number; kind?: string } =>
      isRecord(region) && typeof region.x === "number" && typeof region.y === "number" && typeof region.width === "number" && typeof region.height === "number")
    : [];
  return {
    id: bodyID,
    uri: bodyURI,
    url: bodyURI.startsWith("http") ? bodyURI : `${backdropStudioURL}${bodyURI}`,
    placement: typeof rawBody.placement === "string" ? rawBody.placement : undefined,
    alt_text: typeof rawBody.altText === "string" ? rawBody.altText : undefined,
    reserved_regions: regions,
  };
}
