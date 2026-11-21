export const MAX_PUBLICATION_BYTES = 20 * 1024 * 1024;
export const MAX_PUBLICATION_EXPANDED = 64 * 1024 * 1024;
export const MAX_PUBLICATION_ENTRY = 16 * 1024 * 1024;
export const MAX_PUBLICATION_XML = 2 * 1024 * 1024;
export type PublicationError = "fetch" | "invalid" | "encrypted" | "limit" | "unavailable" | "legacy";
export type PublicationResult = { bytes: ArrayBuffer; xml: Record<string, string> } | { error: PublicationError };
export interface PresentationSlide { title: string; text: string; notes: string; hidden: boolean }
export interface PresentationMetadata { slides: PresentationSlide[]; width: number; height: number; warnings: boolean }
export interface PresentationView { index: number; thumbnails: boolean; zoom: number; labels: { slide: string; hidden: string }; theme: { background: string; surface: string; text: string; muted: string; border: string; accent: string }; rtl: boolean; language: string }
export type PresentationEvent = { type: "ready"; metadata: PresentationMetadata } | { type: "slide"; index: number } | { type: "warning" } | { type: "error"; error: PublicationError } | { type: "boot" };

export function presentationRatio(width: number, height: number): string {
  const ratio = width / height;
  for (const [w, h] of [[16, 9], [4, 3], [16, 10]] as const) if (Math.abs(ratio - w / h) < 0.01) return `${String(w)}:${String(h)}`;
  return `${ratio.toFixed(2)}:1`;
}

export function isPresentationEvent(value: unknown): value is PresentationEvent {
  if (!value || typeof value !== "object" || !("type" in value)) return false;
  const event = value as Record<string, unknown>;
  if (event.type === "boot" || event.type === "warning") return true;
  if (event.type === "error") return "error" in event && ["invalid", "limit", "unavailable"].includes(String(event.error));
  if (event.type === "slide") return Number.isInteger(event.index) && Number(event.index) >= 0 && Number(event.index) < 300;
  if (event.type !== "ready" || !event.metadata || typeof event.metadata !== "object") return false;
  const { slides, width, height, warnings } = event.metadata as Record<string, unknown>;
  if (typeof width !== "number" || !Number.isFinite(width) || width < 1 || width > 10000 || typeof height !== "number" || !Number.isFinite(height) || height < 1 || height > 10000 || typeof warnings !== "boolean" || !Array.isArray(slides) || !slides.length || slides.length > 300) return false;
  return (slides as unknown[]).every(slide => {
    if (!slide || typeof slide !== "object") return false;
    const item = slide as Record<string, unknown>;
    return typeof item.hidden === "boolean" && [item.title, item.text, item.notes].every(text => typeof text === "string" && text.length <= 100000);
  });
}
