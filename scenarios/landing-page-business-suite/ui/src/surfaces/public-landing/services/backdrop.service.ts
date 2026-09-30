export interface BackdropReference {
  id: string;
  uri?: string;
  url?: string;
  placement?: string;
  alt_text?: string;
  reserved_regions?: Array<{ x: number; y: number; width: number; height: number; kind?: string }>;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

/** Resolve only metadata. Raster bytes stay behind the URI owned by Backdrop Studio. */
export async function resolveBackdropReference(
  reference: Pick<BackdropReference, "id">,
  fetcher: typeof fetch = fetch,
): Promise<BackdropReference | null> {
  if (!reference.id) return null;
  let rawBody: unknown;
  try {
    const response = await fetcher(`/api/v1/backdrops/${encodeURIComponent(reference.id)}`, {
      method: "GET",
    });
    if (!response.ok) return null;
    rawBody = await response.json();
  } catch {
    return null;
  }
  if (!isRecord(rawBody) || typeof rawBody.id !== "string" || typeof rawBody.uri !== "string") return null;
  const bodyID = rawBody.id;
  const bodyURI = rawBody.uri;
  const bodyURL = typeof rawBody.url === "string" && rawBody.url !== "" ? rawBody.url : bodyURI;
  const rawRegions = rawBody.reservedRegions;
  const regions = Array.isArray(rawRegions)
    ? rawRegions.filter((region): region is { x: number; y: number; width: number; height: number; kind?: string } =>
      isRecord(region) && typeof region.x === "number" && typeof region.y === "number" && typeof region.width === "number" && typeof region.height === "number")
    : [];
  return {
    id: bodyID,
    uri: bodyURI,
    url: bodyURL,
    placement: typeof rawBody.placement === "string" ? rawBody.placement : undefined,
    alt_text: typeof rawBody.altText === "string" ? rawBody.altText : undefined,
    reserved_regions: regions,
  };
}
