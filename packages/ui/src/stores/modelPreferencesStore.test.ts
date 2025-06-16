import { describe, it, expect, beforeEach } from "vitest";
import { OpenAIModel } from "@vrooli/shared";
import { useModelPreferencesStore } from "./modelPreferencesStore.js";

describe("modelPreferencesStore", () => {
    beforeEach(() => {
        // Clear the store before each test
        useModelPreferencesStore.getState().clearPreferredModel();
    });

    it("should initialize with null preferred model", () => {
        const { preferredModel } = useModelPreferencesStore.getState();
        expect(preferredModel).toBeNull();
    });

    it("should set and get preferred model", () => {
        const testModel = OpenAIModel.Gpt4o_Mini;
        useModelPreferencesStore.getState().setPreferredModel(testModel);
        
        const { preferredModel } = useModelPreferencesStore.getState();
        expect(preferredModel).toBe(testModel);
    });

    it("should clear preferred model", () => {
        const testModel = OpenAIModel.Gpt4o_Mini;
        useModelPreferencesStore.getState().setPreferredModel(testModel);
        useModelPreferencesStore.getState().clearPreferredModel();
        
        const { preferredModel } = useModelPreferencesStore.getState();
        expect(preferredModel).toBeNull();
    });

    it("should validate and set model when valid", () => {
        const testModel = OpenAIModel.Gpt4o;
        const availableModels = [OpenAIModel.Gpt4o_Mini, OpenAIModel.Gpt4o];
        
        const result = useModelPreferencesStore.getState().validateAndSetPreferredModel(testModel, availableModels);
        
        expect(result).toBe(true);
        const { preferredModel } = useModelPreferencesStore.getState();
        expect(preferredModel).toBe(testModel);
    });

    it("should not set model when invalid", () => {
        const testModel = OpenAIModel.Gpt4o;
        const availableModels = [OpenAIModel.Gpt4o_Mini]; // Missing Gpt4o
        
        const result = useModelPreferencesStore.getState().validateAndSetPreferredModel(testModel, availableModels);
        
        expect(result).toBe(false);
        const { preferredModel } = useModelPreferencesStore.getState();
        expect(preferredModel).toBeNull(); // Should remain null
    });
});