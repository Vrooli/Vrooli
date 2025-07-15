import { describe, it, expect, beforeEach, vi } from "vitest";
import { OpenAIModel, type AIServicesInfo } from "@vrooli/shared";
import { getPreferredModel, getPreferredAvailableModel, storageModelToAvailableModel, availableModelToStorageModel, type AvailableModel } from "./ai.js";
import { useModelPreferencesStore } from "../stores/modelPreferencesStore.js";

// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-06-18
// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-01 - Fixed 7 'as any' type assertions with proper Storage and OpenAIModel types

// Constants from ai.ts - we can't import them directly due to module boundaries
const AI_SERVICE_CACHE_KEY = "AI_SERVICE_CACHE";

// Mock localStorage
const localStorageMock = {
    getItem: vi.fn(),
    setItem: vi.fn(),
    removeItem: vi.fn(),
    clear: vi.fn(),
};
global.localStorage = localStorageMock as Storage;

// Mock service info for testing
const mockServiceInfo: AIServicesInfo = {
    defaultService: "openai",
    services: {
        openai: {
            enabled: true,
            defaultModel: OpenAIModel.Gpt4o_Mini,
            models: {},
        },
    },
};

// Mock available models for testing
const mockAvailableModels: AvailableModel[] = [
    {
        name: "GPT_4o_Mini_Name",
        description: "GPT_4o_Mini_Description_Short", 
        value: OpenAIModel.Gpt4o_Mini,
    },
    {
        name: "GPT_4o_Name",
        description: "GPT_4o_Description_Short",
        value: OpenAIModel.Gpt4o,
    },
];

describe("model conversion utilities", () => {
    it("should convert AvailableModel to storage model", () => {
        const result = availableModelToStorageModel(mockAvailableModels[0]);
        expect(result).toBe(OpenAIModel.Gpt4o_Mini);
    });

    it("should convert storage model back to AvailableModel", () => {
        const result = storageModelToAvailableModel(OpenAIModel.Gpt4o, mockAvailableModels);
        expect(result).toEqual(mockAvailableModels[1]);
    });

    it("should return null for invalid storage model", () => {
        const result = storageModelToAvailableModel("invalid-model" as OpenAIModel, mockAvailableModels);
        expect(result).toBeNull();
    });
});

describe("getPreferredAvailableModel", () => {
    beforeEach(() => {
        // Clear mocks and store
        vi.clearAllMocks();
        useModelPreferencesStore.getState().clearPreferredModel();
        localStorage.clear();
    });

    it("should return user's preferred model when valid", () => {
        const preferredModel = OpenAIModel.Gpt4o;
        useModelPreferencesStore.getState().setPreferredModel(preferredModel);
        
        const result = getPreferredAvailableModel(mockAvailableModels);
        expect(result).toEqual(mockAvailableModels[1]);
    });

    it("should clear invalid preference and return first available", () => {
        const invalidModel = "invalid-model" as OpenAIModel;
        useModelPreferencesStore.getState().setPreferredModel(invalidModel);
        
        const result = getPreferredAvailableModel(mockAvailableModels);
        expect(result).toEqual(mockAvailableModels[0]);
        
        // Should have cleared the invalid preference
        const storedValue = useModelPreferencesStore.getState().preferredModel;
        expect(storedValue).toBeNull();
    });

    it("should return first available when no preference set", () => {
        const result = getPreferredAvailableModel(mockAvailableModels);
        expect(result).toEqual(mockAvailableModels[0]);
    });

    it("should return null when no models available", () => {
        const result = getPreferredAvailableModel([]);
        expect(result).toBeNull();
    });
});

describe("getPreferredModel (legacy)", () => {
    beforeEach(() => {
        // Clear mocks and store
        vi.clearAllMocks();
        useModelPreferencesStore.getState().clearPreferredModel();
        localStorage.clear();
    });

    it("should return user's preferred model when set and available", () => {
        const preferredModel = OpenAIModel.Gpt4o; // This is "gpt-4o-2024-05-13"
        
        // Mock the service config to include the preferred model
        const mockServiceConfig: AIServicesInfo = {
            ...mockServiceInfo,
            services: {
                openai: {
                    enabled: true,
                    defaultModel: OpenAIModel.Gpt4o_Mini,
                    models: {
                        [OpenAIModel.Gpt4o]: {
                            enabled: true,
                            name: "GPT_4o_Name" as const,
                            descriptionShort: "GPT_4o_Description_Short" as const,
                        },
                        [OpenAIModel.Gpt4o_Mini]: {
                            enabled: true,
                            name: "GPT_4o_Mini_Name" as const,
                            descriptionShort: "GPT_4o_Mini_Description_Short" as const,
                        },
                    },
                },
            },
        };
        
        // Mock localStorage to return the service config
        localStorageMock.getItem.mockImplementation((key) => {
            if (key === AI_SERVICE_CACHE_KEY) {
                return JSON.stringify({
                    config: mockServiceConfig,
                    timestamp: Date.now(),
                });
            }
            return null;
        });
        
        useModelPreferencesStore.getState().setPreferredModel(preferredModel);
        
        const result = getPreferredModel();
        expect(result).toBe(preferredModel);
    });

    it("should return gpt-4o-mini as fallback when no preference is set", () => {
        // No preference set, no service config available
        const result = getPreferredModel();
        expect(result).toBe("gpt-4o-mini");
    });
});
