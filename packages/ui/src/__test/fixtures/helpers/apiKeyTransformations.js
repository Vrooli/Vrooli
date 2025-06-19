import { generatePK } from "@vrooli/shared";

/**
 * Mock API Key service for round-trip testing
 * Simulates database storage and API operations
 */

// In-memory storage for test API keys
let apiKeyStorage = {};

export const mockApiKeyService = {
    /**
     * Create a new API key
     */
    async create(apiKeyData) {
        const now = new Date().toISOString();
        const apiKey = {
            __typename: "ApiKey",
            ...apiKeyData,
            creditsUsed: "0",
            disabledAt: apiKeyData.disabled ? now : null,
            createdAt: now,
            updatedAt: now,
        };
        
        // Store in our mock database
        apiKeyStorage[apiKey.id] = apiKey;
        
        // For create operation, return with the generated key
        return {
            ...apiKey,
            key: `sk_test_${generatePK()}_${Math.random().toString(36).substring(2, 15)}`,
        };
    },
    
    /**
     * Find API key by ID
     */
    async findById(id) {
        const apiKey = apiKeyStorage[id];
        if (!apiKey) {
            throw new Error(`API key with id ${id} not found`);
        }
        return { ...apiKey };
    },
    
    /**
     * Update an existing API key
     */
    async update(id, updateData) {
        const existing = apiKeyStorage[id];
        if (!existing) {
            throw new Error(`API key with id ${id} not found`);
        }
        
        const now = new Date().toISOString();
        const updated = {
            ...existing,
            ...updateData,
            updatedAt: now,
        };
        
        // Handle disabled state
        if (updateData.disabled !== undefined) {
            updated.disabledAt = updateData.disabled ? now : null;
        }
        
        // Store the updated API key
        apiKeyStorage[id] = updated;
        
        return { ...updated };
    },
    
    /**
     * Delete an API key
     */
    async delete(id) {
        if (!apiKeyStorage[id]) {
            throw new Error(`API key with id ${id} not found`);
        }
        
        delete apiKeyStorage[id];
        
        return { success: true };
    },
    
    /**
     * Clear all test data (useful for test cleanup)
     */
    clearAll() {
        apiKeyStorage = {};
    }
};

// Make storage accessible for test setup/teardown
Object.defineProperty(globalThis, '__testApiKeyStorage', {
    get: () => apiKeyStorage,
    set: (value) => { apiKeyStorage = value; },
    configurable: true,
});