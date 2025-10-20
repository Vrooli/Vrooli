const DATA_URL_PATTERN = /^data:/i;
const PROTOCOL_PATTERN = /^[a-z][a-z0-9+.-]*:/i;

export const ensureDataUrl = (
  value: string | null | undefined,
  mimeType = 'image/png',
): string | null => {
  if (!value) {
    return null;
  }

  const trimmed = value.trim();
  if (!trimmed) {
    return null;
  }

  if (DATA_URL_PATTERN.test(trimmed)) {
    return trimmed;
  }

  if (PROTOCOL_PATTERN.test(trimmed)) {
    return trimmed;
  }

  return `data:${mimeType};base64,${trimmed}`;
};

export default ensureDataUrl;
