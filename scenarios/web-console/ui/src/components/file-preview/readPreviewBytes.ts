// Office readers share the same bounded byte transport inside their workers.
export async function readPreviewBytes(href: string, maxBytes: number, signal?: AbortSignal): Promise<ArrayBuffer> {
  const response = await fetch(href, { credentials: "same-origin", signal });
  if (!response.ok || !response.body) { await response.body?.cancel(); throw new Error("fetch"); }
  if (Number(response.headers.get("content-length")) > maxBytes) {
    await response.body.cancel(); throw new RangeError("bytes");
  }
  const reader = response.body.getReader(), chunks: Uint8Array[] = [];
  let size = 0;
  try {
    for (;;) {
      const { value, done } = await reader.read();
      if (done) break;
      size += value.byteLength;
      if (size > maxBytes) { await reader.cancel(); throw new RangeError("bytes"); }
      chunks.push(value);
    }
  } finally { reader.releaseLock(); }
  const bytes = new Uint8Array(size); let offset = 0;
  for (const chunk of chunks) { bytes.set(chunk, offset); offset += chunk.length; }
  return bytes.buffer;
}
