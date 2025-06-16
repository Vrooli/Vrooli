import type { LlmServiceModel } from "@vrooli/shared";
import { create } from "zustand";
import { persist } from "zustand/middleware";

interface ModelPreferencesState {
    /**
     * The user's preferred model for AI interactions
     */
    preferredModel: LlmServiceModel | null;
    /**
     * Sets the user's preferred model
     */
    setPreferredModel: (model: LlmServiceModel) => void;
    /**
     * Clears the user's model preference
     */
    clearPreferredModel: () => void;
    /**
     * Validates and sets the preferred model only if it's valid
     * @param model The model value to validate and set
     * @param availableModelValues Array of valid model values
     */
    validateAndSetPreferredModel: (model: LlmServiceModel, availableModelValues: LlmServiceModel[]) => boolean;
}

/**
 * Validates that a model is in the list of available models
 */
function validateModel(model: LlmServiceModel, availableModels: LlmServiceModel[]): boolean {
    return availableModels.includes(model);
}

/**
 * Store for managing the user's AI model preferences.
 * Persists to localStorage to maintain preferences across sessions.
 */
export const useModelPreferencesStore = create<ModelPreferencesState>()(
    persist(
        (set, get) => ({
            preferredModel: null,
            setPreferredModel: (model) => set({ preferredModel: model }),
            clearPreferredModel: () => set({ preferredModel: null }),
            validateAndSetPreferredModel: (model, availableModelValues) => {
                if (validateModel(model, availableModelValues)) {
                    set({ preferredModel: model });
                    return true;
                }
                return false;
            },
        }),
        {
            name: "model-preferences",
        },
    ),
);