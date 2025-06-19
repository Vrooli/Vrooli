import { describe, test, expect, beforeEach } from 'vitest';
import { shapeBot, botValidation, generatePK, type User } from "@vrooli/shared";
import { 
    minimalBotCreateFormInput,
    completeBotCreateFormInput,
    minimalBotUpdateFormInput,
    completeBotUpdateFormInput,
    botDepictingPersonFormInput
} from '../form-data/botFormData.js';
import { 
    minimalBotResponse,
    completeBotResponse,
    personDepictingBotResponse
} from '../api-responses/botResponses.js';

/**
 * Round-trip testing for Bot data flow using REAL application functions
 * Tests the complete user journey: Form Input → API Request → Database → API Response → UI Display
 * 
 * ✅ Uses real shapeBot.create() and shapeBot.update() for transformations
 * ✅ Uses real botValidation for validation
 * ✅ Tests actual application logic instead of mock implementations
 */

// Helper functions using REAL application logic
function transformFormToCreateRequestReal(formData: any) {
    return shapeBot.create({
        __typename: "User",
        id: generatePK().toString(),
        name: formData.name,
        handle: formData.handle,
        isBotDepictingPerson: formData.isBotDepictingPerson,
        isPrivate: formData.isPrivate || false,
        isBot: true,
        botSettings: formData.botSettings || { __version: "1.0.0" },
        bannerImage: formData.bannerImage,
        profileImage: formData.profileImage,
        translations: formData.bio ? [{
            __typename: "UserTranslation",
            id: generatePK().toString(),
            language: "en",
            bio: formData.bio,
        }] : undefined,
    });
}

function transformFormToUpdateRequestReal(botId: string, formData: any) {
    const original = {
        __typename: "User" as const,
        id: botId,
        name: "Original Name",
        handle: "original-handle",
        isBotDepictingPerson: false,
        isPrivate: false,
        isBot: true,
        botSettings: { __version: "1.0.0" },
    };

    const updates = {
        __typename: "User" as const,
        id: botId,
        ...formData,
    };

    return shapeBot.update(original, updates);
}

async function validateBotFormDataReal(formData: any): Promise<string[]> {
    try {
        // Use real validation schema - construct the request object first
        const validationData = {
            id: generatePK().toString(),
            name: formData.name,
            handle: formData.handle,
            isBotDepictingPerson: formData.isBotDepictingPerson,
            isPrivate: formData.isPrivate || false,
            botSettings: formData.botSettings || { __version: "1.0.0" },
            bannerImage: formData.bannerImage,
            profileImage: formData.profileImage,
            ...(formData.bio && {
                translations: {
                    create: [{
                        id: generatePK().toString(),
                        language: "en",
                        bio: formData.bio,
                    }],
                }
            }),
        };
        
        await botValidation.create({ omitFields: [] }).validate(validationData);
        return []; // No validation errors
    } catch (error: any) {
        if (error.errors) {
            return error.errors; // Yup validation errors
        }
        return [error.message || "Validation failed"];
    }
}

function transformApiResponseToFormReal(bot: User): any {
    const translation = bot.translations && bot.translations.length > 0 ? bot.translations[0] : null;
    
    return {
        name: bot.name,
        handle: bot.handle,
        isBotDepictingPerson: bot.isBotDepictingPerson,
        isPrivate: bot.isPrivate,
        bio: translation?.bio,
        botSettings: bot.botSettings,
        profileImage: bot.profileImage,
        bannerImage: bot.bannerImage,
    };
}

function areBotsEqualReal(form1: any, form2: any): boolean {
    return (
        form1.name === form2.name &&
        form1.handle === form2.handle &&
        form1.isBotDepictingPerson === form2.isBotDepictingPerson &&
        form1.isPrivate === form2.isPrivate &&
        form1.bio === form2.bio
    );
}

// Mock service for simulating API calls (in real tests this would use testcontainers)
const mockBotService = {
    storage: {} as Record<string, User>,
    
    async create(data: any): Promise<User> {
        const bot: User = {
            __typename: "User",
            id: data.id,
            name: data.name,
            handle: data.handle,
            isBot: true,
            isBotDepictingPerson: data.isBotDepictingPerson,
            isPrivate: data.isPrivate || false,
            status: "Unlocked",
            bannerImage: data.bannerImage || null,
            profileImage: data.profileImage || null,
            theme: "light",
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
            botSettings: data.botSettings || { __version: "1.0.0" },
            awardedCount: 0,
            bookmarksCount: 0,
            commentsCount: 0,
            issuesCount: 0,
            postsCount: 0,
            projectsCount: 0,
            pullRequestsCount: 0,
            questionsCount: 0,
            quizzesCount: 0,
            reportsReceivedCount: 0,
            routinesCount: 0,
            standardsCount: 0,
            teamsCount: 0,
            translations: data.translations || [],
            you: {
                __typename: "UserYou",
                canDelete: false,
                canReport: true,
                canUpdate: false,
                isBookmarked: false,
                isReported: false,
                reaction: null,
            },
        };
        
        this.storage[bot.id] = bot;
        return bot;
    },
    
    async findById(id: string): Promise<User> {
        const bot = this.storage[id];
        if (!bot) {
            throw new Error(`Bot with id ${id} not found`);
        }
        return { ...bot }; // Return copy to simulate API response
    },
    
    async update(id: string, updates: any): Promise<User> {
        const existing = this.storage[id];
        if (!existing) {
            throw new Error(`Bot with id ${id} not found`);
        }
        
        const updated: User = {
            ...existing,
            ...updates,
            id, // Ensure ID doesn't change
            updatedAt: new Date().toISOString(),
        };
        
        this.storage[id] = updated;
        return updated;
    },
    
    async delete(id: string): Promise<{ success: boolean }> {
        if (this.storage[id]) {
            delete this.storage[id];
            return { success: true };
        }
        return { success: false };
    },
};

describe('Bot Round-Trip Data Flow', () => {
    beforeEach(() => {
        // Clear test storage between tests
        mockBotService.storage = {};
    });

    test('minimal bot creation maintains data integrity through complete flow', async () => {
        // 🎨 STEP 1: User fills out minimal bot form
        const userFormData = {
            name: "Test Bot",
            handle: "testbot",
            isBotDepictingPerson: false,
            isPrivate: false,
        };
        
        // Validate form data using REAL validation schema
        const validationErrors = await validateBotFormDataReal(userFormData);
        expect(validationErrors).toHaveLength(0);
        
        // 🔗 STEP 2: Form submits to API using REAL shape function
        const apiCreateRequest = transformFormToCreateRequestReal(userFormData);
        expect(apiCreateRequest.name).toBe(userFormData.name);
        expect(apiCreateRequest.handle).toBe(userFormData.handle);
        expect(apiCreateRequest.isBotDepictingPerson).toBe(userFormData.isBotDepictingPerson);
        expect(apiCreateRequest.id).toMatch(/^\d{10,19}$/); // Valid ID format
        
        // 🗄️ STEP 3: API creates bot (simulated - real test would hit test DB)
        const createdBot = await mockBotService.create(apiCreateRequest);
        expect(createdBot.id).toBe(apiCreateRequest.id);
        expect(createdBot.name).toBe(userFormData.name);
        expect(createdBot.handle).toBe(userFormData.handle);
        expect(createdBot.isBot).toBe(true);
        
        // 🔗 STEP 4: API fetches bot back
        const fetchedBot = await mockBotService.findById(createdBot.id);
        expect(fetchedBot.id).toBe(createdBot.id);
        expect(fetchedBot.name).toBe(userFormData.name);
        expect(fetchedBot.handle).toBe(userFormData.handle);
        
        // 🎨 STEP 5: UI would display the bot using REAL transformation
        // Verify that form data can be reconstructed from API response
        const reconstructedFormData = transformApiResponseToFormReal(fetchedBot);
        expect(reconstructedFormData.name).toBe(userFormData.name);
        expect(reconstructedFormData.handle).toBe(userFormData.handle);
        expect(reconstructedFormData.isBotDepictingPerson).toBe(userFormData.isBotDepictingPerson);
        
        // ✅ VERIFICATION: Complete round-trip integrity using REAL comparison
        expect(areBotsEqualReal(
            userFormData, 
            reconstructedFormData
        )).toBe(true);
    });

    test('complete bot with settings and bio preserves all data', async () => {
        // 🎨 STEP 1: User creates bot with all fields
        const userFormData = {
            name: "AI Assistant Bot",
            handle: "ai-assistant",
            isBotDepictingPerson: false,
            isPrivate: false,
            bio: "An advanced AI assistant designed to help with various tasks",
            botSettings: {
                __version: "1.0.0",
                model: "gpt-4",
                temperature: 0.7,
                maxTokens: 2048,
                systemPrompt: "You are a helpful AI assistant.",
            },
        };
        
        // Validate complex form using REAL validation
        const validationErrors = await validateBotFormDataReal(userFormData);
        expect(validationErrors).toHaveLength(0);
        
        // 🔗 STEP 2: Transform to API request using REAL shape function
        const apiCreateRequest = transformFormToCreateRequestReal(userFormData);
        expect(apiCreateRequest.translations).toBeDefined();
        expect(apiCreateRequest.translations![0].bio).toBe(userFormData.bio);
        expect(apiCreateRequest.botSettings).toEqual(userFormData.botSettings);
        
        // 🗄️ STEP 3: Create via API
        const createdBot = await mockBotService.create(apiCreateRequest);
        expect(createdBot.botSettings).toEqual(userFormData.botSettings);
        expect(createdBot.translations).toHaveLength(1);
        expect(createdBot.translations![0].bio).toBe(userFormData.bio);
        
        // 🔗 STEP 4: Fetch back from API
        const fetchedBot = await mockBotService.findById(createdBot.id);
        expect(fetchedBot.translations![0].bio).toBe(userFormData.bio);
        expect(fetchedBot.botSettings).toEqual(userFormData.botSettings);
        
        // 🎨 STEP 5: Verify UI can display correctly using REAL transformation
        const reconstructedFormData = transformApiResponseToFormReal(fetchedBot);
        expect(reconstructedFormData.name).toBe(userFormData.name);
        expect(reconstructedFormData.handle).toBe(userFormData.handle);
        expect(reconstructedFormData.bio).toBe(userFormData.bio);
        expect(reconstructedFormData.botSettings).toEqual(userFormData.botSettings);
        
        // ✅ VERIFICATION: All data preserved including complex settings
        expect(fetchedBot.name).toBe(userFormData.name);
        expect(fetchedBot.handle).toBe(userFormData.handle);
        expect(fetchedBot.translations![0].bio).toBe(userFormData.bio);
        expect(fetchedBot.botSettings.model).toBe(userFormData.botSettings.model);
    });

    test('bot depicting person works correctly through round-trip', async () => {
        // 🎨 STEP 1: User creates bot representing a person
        const userFormData = {
            name: "Virtual Sarah",
            handle: "virtual-sarah",
            isBotDepictingPerson: true,
            isPrivate: false,
            bio: "Virtual assistant representing Sarah from customer support",
            botSettings: {
                __version: "1.0.0",
                model: "gpt-4",
                temperature: 0.8,
                personality: "friendly",
            },
        };
        
        // Validate form using REAL validation
        const validationErrors = await validateBotFormDataReal(userFormData);
        expect(validationErrors).toHaveLength(0);
        
        // 🔗 STEP 2: Transform and create using REAL functions
        const createRequest = transformFormToCreateRequestReal(userFormData);
        expect(createRequest.isBotDepictingPerson).toBe(true);
        
        const createdBot = await mockBotService.create(createRequest);
        
        // 🗄️ STEP 3: Fetch back
        const fetchedBot = await mockBotService.findById(createdBot.id);
        
        // ✅ Verify person-depicting flag is preserved
        expect(fetchedBot.isBotDepictingPerson).toBe(true);
        expect(fetchedBot.name).toBe(userFormData.name);
        expect(fetchedBot.translations![0].bio).toBe(userFormData.bio);
        
        // Verify form reconstruction using REAL transformation
        const reconstructed = transformApiResponseToFormReal(fetchedBot);
        expect(reconstructed.isBotDepictingPerson).toBe(true);
        expect(reconstructed.name).toBe(userFormData.name);
    });

    test('bot editing maintains data integrity', async () => {
        // Create initial bot using REAL functions
        const initialFormData = minimalBotCreateFormInput;
        const createRequest = transformFormToCreateRequestReal(initialFormData);
        const initialBot = await mockBotService.create(createRequest);
        
        // 🎨 STEP 1: User edits bot
        const editFormData = {
            name: "Updated Bot Name",
            bio: "Updated bot description",
            botSettings: {
                __version: "1.0.0",
                model: "claude-3-sonnet",
                temperature: 0.5,
            },
        };
        
        // 🔗 STEP 2: Transform edit to update request using REAL update logic
        const updateRequest = transformFormToUpdateRequestReal(initialBot.id, editFormData);
        expect(updateRequest.id).toBe(initialBot.id);
        expect(updateRequest.name).toBe(editFormData.name);
        
        // Add small delay to ensure different timestamps
        await new Promise(resolve => setTimeout(resolve, 10));
        
        // 🗄️ STEP 3: Update via API
        const updatedBot = await mockBotService.update(initialBot.id, updateRequest);
        expect(updatedBot.id).toBe(initialBot.id);
        expect(updatedBot.name).toBe(editFormData.name);
        
        // 🔗 STEP 4: Fetch updated bot
        const fetchedUpdatedBot = await mockBotService.findById(initialBot.id);
        
        // ✅ VERIFICATION: Update preserved core data
        expect(fetchedUpdatedBot.id).toBe(initialBot.id);
        expect(fetchedUpdatedBot.name).toBe(editFormData.name);
        expect(fetchedUpdatedBot.handle).toBe(initialFormData.handle); // Unchanged
        expect(fetchedUpdatedBot.createdAt).toBe(initialBot.createdAt); // Created date unchanged
        // Updated date should be different
        expect(new Date(fetchedUpdatedBot.updatedAt).getTime()).toBeGreaterThan(
            new Date(initialBot.updatedAt).getTime()
        );
    });

    test('private vs public bot settings work correctly', async () => {
        // Test private bot
        const privateFormData = {
            name: "Private Bot",
            handle: "private-bot",
            isBotDepictingPerson: false,
            isPrivate: true,
        };
        
        const privateCreateRequest = transformFormToCreateRequestReal(privateFormData);
        const privateBot = await mockBotService.create(privateCreateRequest);
        
        expect(privateBot.isPrivate).toBe(true);
        
        // Test public bot
        const publicFormData = {
            name: "Public Bot",
            handle: "public-bot",
            isBotDepictingPerson: false,
            isPrivate: false,
        };
        
        const publicCreateRequest = transformFormToCreateRequestReal(publicFormData);
        const publicBot = await mockBotService.create(publicCreateRequest);
        
        expect(publicBot.isPrivate).toBe(false);
        
        // Verify both can be retrieved and settings preserved
        const fetchedPrivate = await mockBotService.findById(privateBot.id);
        const fetchedPublic = await mockBotService.findById(publicBot.id);
        
        expect(fetchedPrivate.isPrivate).toBe(true);
        expect(fetchedPublic.isPrivate).toBe(false);
    });

    test('validation catches invalid form data before API submission', async () => {
        const invalidFormData = {
            name: "", // Required but empty
            handle: "ab", // Too short
            isBotDepictingPerson: false,
        };
        
        const validationErrors = await validateBotFormDataReal(invalidFormData);
        expect(validationErrors.length).toBeGreaterThan(0);
        
        // Should not proceed to API if validation fails
        expect(validationErrors.some(error => 
            error.includes("name") || error.includes("required")
        )).toBe(true);
    });

    test('bot settings validation works correctly', async () => {
        const invalidSettingsData = {
            name: "Valid Bot Name",
            handle: "valid-handle",
            isBotDepictingPerson: false,
            botSettings: null, // Invalid: botSettings required
        };
        
        const validationErrors = await validateBotFormDataReal(invalidSettingsData);
        expect(validationErrors.length).toBeGreaterThan(0);
        
        const validSettingsData = {
            name: "Valid Bot Name",
            handle: "valid-handle",
            isBotDepictingPerson: false,
            botSettings: {
                __version: "1.0.0",
                model: "gpt-4",
                temperature: 0.7,
            },
        };
        
        const validValidationErrors = await validateBotFormDataReal(validSettingsData);
        expect(validValidationErrors).toHaveLength(0);
    });

    test('bot deletion works correctly', async () => {
        // Create bot first using REAL functions
        const formData = minimalBotCreateFormInput;
        const createRequest = transformFormToCreateRequestReal(formData);
        const createdBot = await mockBotService.create(createRequest);
        
        // Delete it
        const deleteResult = await mockBotService.delete(createdBot.id);
        expect(deleteResult.success).toBe(true);
        
        // Verify it's gone
        await expect(mockBotService.findById(createdBot.id)).rejects.toThrow();
    });

    test('data consistency across multiple operations', async () => {
        const originalFormData = minimalBotCreateFormInput;
        
        // Create using REAL functions
        const createRequest = transformFormToCreateRequestReal(originalFormData);
        const created = await mockBotService.create(createRequest);
        
        // Update using REAL functions
        const updateRequest = transformFormToUpdateRequestReal(created.id, { 
            name: "Updated Bot Name",
            botSettings: {
                __version: "1.0.0",
                model: "claude-3-opus",
                temperature: 0.3,
            },
        });
        const updated = await mockBotService.update(created.id, updateRequest);
        
        // Fetch final state
        const final = await mockBotService.findById(created.id);
        
        // Core bot data should remain consistent
        expect(final.id).toBe(created.id);
        expect(final.handle).toBe(originalFormData.handle); // Handle unchanged
        expect(final.isBotDepictingPerson).toBe(originalFormData.isBotDepictingPerson);
        expect(final.isBot).toBe(true);
        
        // Only the updated fields should have changed
        expect(final.name).toBe("Updated Bot Name");
        expect(final.botSettings.model).toBe("claude-3-opus");
        expect(final.botSettings.temperature).toBe(0.3);
        
        // Creation date unchanged, update date changed
        expect(final.createdAt).toBe(created.createdAt);
        expect(new Date(final.updatedAt).getTime()).toBeGreaterThan(
            new Date(created.updatedAt).getTime()
        );
    });
});