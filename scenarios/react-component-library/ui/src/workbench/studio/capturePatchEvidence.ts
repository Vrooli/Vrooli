export async function readAsArrayBuffer(blob: Blob): Promise<ArrayBuffer> {
  return new Promise<ArrayBuffer>((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      if (reader.result instanceof ArrayBuffer) resolve(reader.result);
      else reject(new Error("capturePatch.imageReadFailed"));
    };
    reader.onerror = () => reject(new Error("capturePatch.imageReadFailed"));
    reader.readAsArrayBuffer(blob);
  });
}

export async function readAsText(blob: Blob): Promise<string> {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      if (typeof reader.result === "string") resolve(reader.result);
      else reject(new Error("capturePatch.patchReadFailed"));
    };
    reader.onerror = () => reject(new Error("capturePatch.patchReadFailed"));
    reader.readAsText(blob);
  });
}

export async function sha256(file: Blob): Promise<string> {
  try {
    const bytes = await readAsArrayBuffer(file);
    const digest = await globalThis.crypto.subtle.digest("SHA-256", bytes);
    return [...new Uint8Array(digest)].map((byte) => byte.toString(16).padStart(2, "0")).join("");
  } catch {
    throw new Error("capturePatch.imageReadFailed");
  }
}
