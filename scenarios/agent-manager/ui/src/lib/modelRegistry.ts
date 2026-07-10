// Model registry helper functions

import type { ModelOption, ModelRegistry, PresetChain } from "../types";

export type EditableModelOption = { id: string; description: string };

export function normalizeModelOption(option: ModelOption): EditableModelOption {
  if (typeof option === "string") {
    return { id: option, description: "" };
  }
  return {
    id: option.id,
    description: option.description ?? "",
  };
}

export function normalizeModelOptions(options: ModelOption[]): EditableModelOption[] {
  return options.map(normalizeModelOption);
}

/**
 * presetPrimaryMap collapses a preset chain map into a `preset -> primary model ID`
 * map for UI widgets that present a single representative model per preset (e.g.
 * ModelConfigSelector). The full chain remains available via the registry itself;
 * this helper is purely for display.
 */
export function presetPrimaryMap(
  presets: Record<string, PresetChain> | undefined
): Record<string, string> {
  const out: Record<string, string> = {};
  if (!presets) return out;
  for (const [key, chain] of Object.entries(presets)) {
    if (!Array.isArray(chain)) continue;
    for (const entry of chain) {
      const trimmed = entry?.trim();
      if (trimmed) {
        out[key] = trimmed;
        break;
      }
    }
  }
  return out;
}

/**
 * sanitizeChain preserves legacy registry fixture invariants while remaining
 * consumers migrate to the read-only declared policy catalog:
 *   - Concrete entries are trimmed.
 *   - Duplicate concrete entries are collapsed to the first occurrence.
 *   - An empty-string sentinel is kept only if it is the last entry.
 *   - A chain with no concrete entries is dropped (invalid on the backend).
 */
export function sanitizeChain(chain: PresetChain): PresetChain {
  const seen = new Set<string>();
  const out: PresetChain = [];
  let hasSentinel = false;
  for (let i = 0; i < chain.length; i += 1) {
    const entry = chain[i];
    if (entry === undefined) continue;
    if (entry === "") {
      // Only preserve the sentinel if it is the final entry; anywhere else
      // it is invalid and dropped.
      if (i === chain.length - 1) {
        hasSentinel = true;
      }
      continue;
    }
    const trimmed = entry.trim();
    if (!trimmed || seen.has(trimmed)) continue;
    seen.add(trimmed);
    out.push(trimmed);
  }
  if (hasSentinel && out.length > 0) {
    out.push("");
  }
  return out;
}

export function sanitizeModelRegistry(registry: ModelRegistry): ModelRegistry {
  const runners: ModelRegistry["runners"] = {};
  const fallbackRunnerTypes = Array.from(
    new Set(
      (registry.fallbackRunnerTypes ?? [])
        .map((runner) => runner.trim())
        .filter((runner) => runner.length > 0)
    )
  );
  for (const [runnerKey, runner] of Object.entries(registry.runners)) {
    const normalizedModels = normalizeModelOptions(runner.models)
      .map((model) => ({
        id: model.id.trim(),
        description: model.description.trim(),
      }))
      .filter((model) => model.id.length > 0);
    const models: ModelOption[] = normalizedModels.map((model) =>
      model.description ? { id: model.id, description: model.description } : model.id
    );
    const presets: Record<string, PresetChain> = {};
    for (const [presetKey, chain] of Object.entries(runner.presets ?? {})) {
      const sanitized = sanitizeChain(Array.isArray(chain) ? chain : []);
      if (sanitized.length > 0) {
        presets[presetKey] = sanitized;
      }
    }
    runners[runnerKey] = {
      models,
      presets,
    };
  }
  return {
    ...registry,
    fallbackRunnerTypes,
    runners,
  };
}
