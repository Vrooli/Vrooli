import type { Model } from "../../lib/api";

export type SortOption = "name" | "price-asc" | "price-desc" | "context-desc";
export type ModalityFilter = "all" | "text" | "image" | "text+image";

export const SORT_OPTIONS: { value: SortOption; label: string }[] = [
  { value: "name", label: "Name (A-Z)" },
  { value: "price-asc", label: "Price (Low to High)" },
  { value: "price-desc", label: "Price (High to Low)" },
  { value: "context-desc", label: "Context (Largest)" },
];

export const MODALITY_OPTIONS: { value: ModalityFilter; label: string; icon: string }[] = [
  { value: "all", label: "All modalities", icon: "MessageSquare" },
  { value: "text", label: "Text only", icon: "Type" },
  { value: "image", label: "Image support", icon: "Image" },
  { value: "text+image", label: "Text + Image", icon: "Image" },
];

const RECENT_MODELS_KEY = "recentModels";
const MAX_RECENT_MODELS = 5;

/**
 * Get recently used model IDs from localStorage.
 */
export function getRecentModelIds(): string[] {
  try {
    const stored = localStorage.getItem(RECENT_MODELS_KEY);
    if (!stored) return [];
    const parsed: unknown = JSON.parse(stored);
    if (!Array.isArray(parsed)) return [];
    return (parsed as unknown[]).filter((id): id is string => typeof id === "string");
  } catch {
    return [];
  }
}

/**
 * Add a model ID to the recent models list.
 * Moves it to the front if already present, maintains max size.
 */
export function addRecentModelId(modelId: string): void {
  try {
    const current = getRecentModelIds();
    const filtered = current.filter((id) => id !== modelId);
    const updated = [modelId, ...filtered].slice(0, MAX_RECENT_MODELS);
    localStorage.setItem(RECENT_MODELS_KEY, JSON.stringify(updated));
  } catch {
    // Silently fail if localStorage is unavailable
  }
}

/**
 * Remove a model ID from the recent models list.
 */
export function removeRecentModelId(modelId: string): string[] {
  try {
    const current = getRecentModelIds();
    const updated = current.filter((id) => id !== modelId);
    localStorage.setItem(RECENT_MODELS_KEY, JSON.stringify(updated));
    return updated;
  } catch {
    return [];
  }
}

/**
 * Extract provider from model ID or use the provider field.
 */
export function getProvider(model: Model): string {
  if (model.provider) {
    return model.provider;
  }
  const parts = model.id.split("/");
  const firstPart = parts[0];
  if (parts.length >= 2 && firstPart) {
    return firstPart;
  }
  return "other";
}

/**
 * Capitalize provider name.
 */
export function formatProviderName(provider: string): string {
  return provider
    .split(/[-_]/)
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
    .join(" ");
}

/**
 * Format price per million tokens.
 */
export function formatPrice(pricePerToken: number): string {
  const pricePerMillion = pricePerToken * 1_000_000;
  if (pricePerMillion < 0.01) {
    return "<$0.01";
  }
  if (pricePerMillion < 1) {
    return `$${pricePerMillion.toFixed(2)}`;
  }
  return `$${pricePerMillion.toFixed(1)}`;
}

/**
 * Get the combined price for sorting (prompt + completion average).
 */
export function getCombinedPrice(model: Model): number {
  if (!model.pricing) return Infinity;
  return (model.pricing.prompt + model.pricing.completion) / 2;
}

/**
 * Check if model supports a specific input modality.
 */
export function supportsModality(model: Model, modality: string): boolean {
  return model.architecture?.input?.includes(modality) ?? false;
}

/**
 * Get modality badges for a model.
 */
export function getModalityBadges(model: Model): string[] {
  return model.architecture?.input ?? ["text"];
}
