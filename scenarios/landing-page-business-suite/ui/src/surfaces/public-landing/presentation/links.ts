export function safeHref(value?: string): string | undefined {
  if (!value) return undefined;
  for (let index = 0; index < value.length; index++) {
    if (value.charCodeAt(index) <= 32 || value[index] === '\\') return undefined;
  }
  if (value.startsWith('#') || (value.startsWith('/') && !value.startsWith('//'))) return value;
  try { return new URL(value).protocol === 'https:' ? value : undefined; } catch { return undefined; }
}
