// Model registry helper functions

import type { ModelOption, ModelRegistry } from "../types";

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
    const presets: Record<string, string> = {};
    for (const [presetKey, modelId] of Object.entries(runner.presets ?? {})) {
      const trimmedModelId = modelId.trim();
      if (trimmedModelId) {
        presets[presetKey] = trimmedModelId;
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
