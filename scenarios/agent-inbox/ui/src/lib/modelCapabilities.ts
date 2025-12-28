/**
 * Model capability detection for multimodal features.
 *
 * This module determines what input types a model supports based on
 * the architecture.input field from OpenRouter's model metadata.
 */

import type { Model } from "./api";

export interface ModelCapabilities {
  supportsImages: boolean;
  supportsPDFs: boolean;
  supportsText: boolean;
  supportsTools: boolean;
}

/**
 * Known vision-capable model patterns.
 * These are patterns that indicate a model supports image input.
 */
const VISION_MODEL_PATTERNS = [
  "gpt-4o",
  "gpt-4-turbo",
  "gpt-4-vision",
  "claude-3",
  "claude-3.5",
  "gemini-pro-vision",
  "gemini-1.5",
  "gemini-2",
  "llava",
  "vision",
];

/**
 * Known PDF-capable model patterns.
 * These models can process PDF files (via OpenRouter's file-parser plugin).
 * Currently, most vision models also support PDFs through the plugin.
 */
const PDF_MODEL_PATTERNS = [
  "gpt-4o",
  "gpt-4-turbo",
  "claude-3",
  "claude-3.5",
  "gemini-1.5",
  "gemini-2",
];

/**
 * Check if a model supports image input.
 * Uses architecture.input metadata if available, falls back to name pattern matching.
 */
export function supportsImages(model: Model | null): boolean {
  if (!model) return false;

  // Check architecture metadata first
  if (model.architecture?.input) {
    const inputs = model.architecture.input;
    if (inputs.includes("image") || inputs.includes("vision")) {
      return true;
    }
  }

  // Fallback to name-based detection
  const modelId = model.id.toLowerCase();
  return VISION_MODEL_PATTERNS.some(pattern => modelId.includes(pattern.toLowerCase()));
}

/**
 * Check if a model supports PDF input.
 * PDFs are processed via OpenRouter's file-parser plugin.
 */
export function supportsPDFs(model: Model | null): boolean {
  if (!model) return false;

  // PDF support is available for most modern models via the plugin
  // Check architecture metadata first
  if (model.architecture?.input) {
    const inputs = model.architecture.input;
    if (inputs.includes("file") || inputs.includes("document")) {
      return true;
    }
  }

  // Fallback to name-based detection
  const modelId = model.id.toLowerCase();
  return PDF_MODEL_PATTERNS.some(pattern => modelId.includes(pattern.toLowerCase()));
}

/**
 * Check if a model supports tool/function calling.
 * Uses supported_parameters metadata from OpenRouter.
 * This is required for features like web search that use tools under the hood.
 */
export function supportsTools(model: Model | null): boolean {
  if (!model) return false;

  // Check supported_parameters from OpenRouter API
  if (model.supported_parameters) {
    return model.supported_parameters.includes("tools");
  }

  // Fallback: assume no tool support if metadata is missing
  return false;
}

/**
 * Get full capabilities for a model.
 */
export function getModelCapabilities(model: Model | null): ModelCapabilities {
  return {
    supportsImages: supportsImages(model),
    supportsPDFs: supportsPDFs(model),
    supportsText: true, // All models support text
    supportsTools: supportsTools(model),
  };
}

/**
 * Check if any attachments are incompatible with the model.
 * Returns a list of incompatible file types.
 */
export function getIncompatibleAttachments(
  model: Model | null,
  attachments: Array<{ content_type: string }>
): string[] {
  const capabilities = getModelCapabilities(model);
  const incompatible: string[] = [];

  for (const attachment of attachments) {
    const contentType = attachment.content_type;

    if (contentType.startsWith("image/") && !capabilities.supportsImages) {
      if (!incompatible.includes("images")) {
        incompatible.push("images");
      }
    }

    if (contentType === "application/pdf" && !capabilities.supportsPDFs) {
      if (!incompatible.includes("PDFs")) {
        incompatible.push("PDFs");
      }
    }
  }

  return incompatible;
}

/**
 * Check if attachments are compatible with the model.
 */
export function areAttachmentsCompatible(
  model: Model | null,
  attachments: Array<{ content_type: string }>
): boolean {
  return getIncompatibleAttachments(model, attachments).length === 0;
}
