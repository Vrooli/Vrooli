/**
 * Mock Registry
 * 
 * Global registry for managing AI mock behaviors across tests.
 */

import type { LLMRequest } from "@vrooli/shared";
import type { 
    MockBehavior, 
    MockRegistryEntry, 
    AIMockService,
    MockFactoryResult 
} from "../types.js";
import { createAIMockResponse } from "../factories/responseFactory.js";
import { createStreamingMock } from "../factories/streamingFactory.js";
import { validateMockConfig } from "../validators/responseValidator.js";

/**
 * Singleton mock registry
 */
export class MockRegistry implements AIMockService {
    private static instance: MockRegistry;
    private behaviors: Map<string, MockRegistryEntry> = new Map();
    private stats = {
        totalRequests: 0,
        behaviorHits: {} as Record<string, number>
    };
    private capturedInteractions: Array<{ request: LLMRequest; response: MockFactoryResult }> = [];
    private debugMode = false;
    
    private constructor() {}
    
    static getInstance(): MockRegistry {
        if (!MockRegistry.instance) {
            MockRegistry.instance = new MockRegistry();
        }
        return MockRegistry.instance;
    }
    
    /**
     * Register a mock behavior
     */
    registerBehavior(id: string, behavior: MockBehavior): void {
        if (this.debugMode) {
            console.log(`[MockRegistry] Registering behavior: ${id}`);
        }
        
        // Validate behavior config if provided
        if (behavior.response && typeof behavior.response === "object") {
            const validation = validateMockConfig(behavior.response);
            if (!validation.valid) {
                throw new Error(`Invalid mock config for ${id}: ${validation.errors?.join(", ")}`);
            }
        }
        
        this.behaviors.set(id, {
            id,
            behavior,
            uses: 0,
            created: new Date()
        });
        
        this.stats.behaviorHits[id] = 0;
    }
    
    /**
     * Execute a mock request
     */
    async execute(request: LLMRequest): Promise<MockFactoryResult> {
        this.stats.totalRequests++;
        
        if (this.debugMode) {
            console.log(`[MockRegistry] Executing request:`, {
                model: request.model,
                messageCount: request.messages.length,
                hasTools: !!request.tools?.length
            });
        }
        
        // Find matching behavior
        const matchedEntry = this.findMatchingBehavior(request);
        
        if (!matchedEntry) {
            // Use default response
            const defaultResponse = createAIMockResponse({
                content: "Mock response (no matching behavior)",
                model: request.model
            });
            
            this.captureInteraction(request, defaultResponse);
            return defaultResponse;
        }
        
        // Check max uses
        if (matchedEntry.behavior.maxUses && matchedEntry.uses >= matchedEntry.behavior.maxUses) {
            if (this.debugMode) {
                console.log(`[MockRegistry] Behavior ${matchedEntry.id} exhausted (${matchedEntry.uses}/${matchedEntry.behavior.maxUses})`);
            }
            // Behavior exhausted, use default
            const defaultResponse = createAIMockResponse({
                content: "Mock response (behavior exhausted)",
                model: request.model
            });
            
            this.captureInteraction(request, defaultResponse);
            return defaultResponse;
        }
        
        // Generate response
        const config = typeof matchedEntry.behavior.response === "function"
            ? matchedEntry.behavior.response(request)
            : matchedEntry.behavior.response;
        
        // Apply delay if specified
        if (config.delay) {
            await this.sleep(config.delay);
        }
        
        // Update usage stats
        matchedEntry.uses++;
        matchedEntry.lastUsed = new Date();
        this.stats.behaviorHits[matchedEntry.id]++;
        
        // Create response
        const result = createAIMockResponse(config);
        
        // Add streaming if configured
        if (config.streaming) {
            result.stream = createStreamingMock(config.streaming);
        }
        
        this.captureInteraction(request, result);
        
        if (this.debugMode) {
            console.log(`[MockRegistry] Used behavior ${matchedEntry.id}, response:`, result.response);
        }
        
        return result;
    }
    
    /**
     * Clear all behaviors
     */
    clearBehaviors(): void {
        this.behaviors.clear();
        this.stats = {
            totalRequests: 0,
            behaviorHits: {}
        };
        this.capturedInteractions = [];
        
        if (this.debugMode) {
            console.log("[MockRegistry] All behaviors cleared");
        }
    }
    
    /**
     * Get usage statistics
     */
    getStats(): {
        totalRequests: number;
        behaviorHits: Record<string, number>;
        averageLatency: number;
    } {
        return {
            ...this.stats,
            averageLatency: 50 // Mock value
        };
    }
    
    /**
     * Enable/disable debug mode
     */
    setDebugMode(enabled: boolean): void {
        this.debugMode = enabled;
    }
    
    /**
     * Get captured interactions
     */
    getCapturedInteractions(): Array<{ request: LLMRequest; response: MockFactoryResult }> {
        return [...this.capturedInteractions];
    }
    
    /**
     * Clear captured interactions
     */
    clearCapturedInteractions(): void {
        this.capturedInteractions = [];
    }
    
    /**
     * Get all registered behaviors
     */
    getBehaviors(): Map<string, MockRegistryEntry> {
        return new Map(this.behaviors);
    }
    
    /**
     * Update behavior
     */
    updateBehavior(id: string, updates: Partial<MockBehavior>): void {
        const entry = this.behaviors.get(id);
        if (!entry) {
            throw new Error(`Behavior ${id} not found`);
        }
        
        entry.behavior = { ...entry.behavior, ...updates };
        
        if (this.debugMode) {
            console.log(`[MockRegistry] Updated behavior ${id}`);
        }
    }
    
    /**
     * Remove behavior
     */
    removeBehavior(id: string): void {
        const removed = this.behaviors.delete(id);
        if (removed && this.debugMode) {
            console.log(`[MockRegistry] Removed behavior ${id}`);
        }
    }
    
    /**
     * Register multiple behaviors
     */
    registerBehaviors(behaviors: Record<string, MockBehavior>): void {
        Object.entries(behaviors).forEach(([id, behavior]) => {
            this.registerBehavior(id, behavior);
        });
    }
    
    /**
     * Private helper methods
     */
    private findMatchingBehavior(request: LLMRequest): MockRegistryEntry | null {
        const sortedEntries = Array.from(this.behaviors.values())
            .sort((a, b) => (b.behavior.priority || 0) - (a.behavior.priority || 0));
        
        for (const entry of sortedEntries) {
            const { pattern } = entry.behavior;
            
            if (!pattern) {
                // No pattern means always match
                return entry;
            }
            
            if (typeof pattern === "function") {
                if (pattern(request)) {
                    return entry;
                }
            } else if (pattern instanceof RegExp) {
                const text = request.messages
                    .map(m => m.content)
                    .join(" ");
                if (pattern.test(text)) {
                    return entry;
                }
            }
        }
        
        return null;
    }
    
    private captureInteraction(request: LLMRequest, response: MockFactoryResult): void {
        this.capturedInteractions.push({ request, response });
        
        // Limit captured interactions to prevent memory issues
        if (this.capturedInteractions.length > 1000) {
            this.capturedInteractions = this.capturedInteractions.slice(-500);
        }
    }
    
    private sleep(ms: number): Promise<void> {
        return new Promise(resolve => setTimeout(resolve, ms));
    }
}

/**
 * Convenience function to register a behavior on the global registry
 */
export function registerMockBehavior(id: string, behavior: MockBehavior): void {
    MockRegistry.getInstance().registerBehavior(id, behavior);
}

/**
 * Convenience function to clear all behaviors
 */
export function clearAllMockBehaviors(): void {
    MockRegistry.getInstance().clearBehaviors();
}

/**
 * Get the global mock registry instance
 */
export function getMockRegistry(): MockRegistry {
    return MockRegistry.getInstance();
}