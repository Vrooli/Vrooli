import type { Region } from "./region";

export function captureName(assetName: string): string {
  const base = assetName.replace(/\.[^./\\]+$/, "");
  return `capture-${base || "region"}.png`;
}

export function encodeRegion(image: HTMLImageElement, region: Region): Promise<Blob> {
  return new Promise((resolve, reject) => {
    const canvas = document.createElement("canvas");
    canvas.width = region.w;
    canvas.height = region.h;
    const context = canvas.getContext("2d");
    if (!context) {
      reject(new Error("canvas-unavailable"));
      return;
    }
    context.drawImage(image, region.x, region.y, region.w, region.h, 0, 0, region.w, region.h);
    try {
      canvas.toBlob((blob) => {
        if (blob) resolve(blob);
        else reject(new Error("png-encode-failed"));
      }, "image/png");
    } catch (cause) {
      reject(cause instanceof Error ? cause : new Error("png-encode-failed"));
    }
  });
}
