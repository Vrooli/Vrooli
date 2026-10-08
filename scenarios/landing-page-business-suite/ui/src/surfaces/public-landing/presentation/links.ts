export function safeHref(value?: string): string | undefined {
  if (!value) return undefined;
  for (let index = 0; index < value.length; index++) {
    if (value.charCodeAt(index) <= 32 || value.charCodeAt(index) === 127 || value[index] === '\\') return undefined;
  }
  try {
    const decoded = decodeURIComponent(value);
    for (let index = 0; index < decoded.length; index++) {
      if (decoded.charCodeAt(index) < 32 || decoded.charCodeAt(index) === 127 || decoded[index] === '\\') return undefined;
    }
    if (value.startsWith('#')) return value;
    const absolute = /^https:\/\//i.test(value);
    if (!absolute && (!value.startsWith('/') || value.startsWith('//'))) return undefined;
    // Check the original path before URL parsing normalizes dot segments away.
    const path = (absolute ? value.replace(/^https:\/\/[^/?#]*/i, '') : value).split(/[?#]/)[0] ?? '';
    for (const segment of path.split('/')) {
      const part = decodeURIComponent(segment);
      if (part === '.' || part === '..' || part.includes('/') || /%[0-9a-f]{2}/i.test(part)) return undefined;
    }
    if (absolute) {
      const url = new URL(value);
      if (url.protocol !== 'https:' || !url.hostname || url.username || url.password) return undefined;
    }
    return value;
  } catch { return undefined; }
}
