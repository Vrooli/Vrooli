/* c8 ignore start */
/**
 * Tag fixture factory implementation
 * 
 * This demonstrates the pattern for a simple object type with minimal relationships.
 */

import {
    type Tag,
    type TagCreateInput,
    type TagUpdateInput,
    shapeTag,
    tagValidation,
    generatePK,
    tagFixtures as sharedTagFixtures
} from "@vrooli/shared";
import { BaseFormFixtureFactory } from "../BaseFormFixtureFactory.js";
import { BaseRoundTripOrchestrator } from "../BaseRoundTripOrchestrator.js";
import { BaseMSWHandlerFactory } from "../BaseMSWHandlerFactory.js";
import { createValidationAdapter } from "../utils/integration.js";
import type {
    UIFixtureFactory,
    FormFixtureFactory,
    RoundTripOrchestrator,
    MSWHandlerFactory,
    UIStateFixtureFactory,
    ComponentTestUtils,
    TestAPIClient,
    DatabaseVerifier
} from "../types.js";

/**
 * Tag form data type
 * 
 * Tags are simple but support translations for multilingual applications.
 */
export interface TagFormData {
    tag: string;
    description?: string;
    language?: string;
    // UI-specific fields
    isPopular?: boolean;
    suggestedCategory?: string;
}

/**
 * Tag UI state type
 */
export interface TagUIState {
    isLoading: boolean;
    tags: Tag[];
    selectedTags: string[];
    suggestions: Tag[];
    error: string | null;
}

/**
 * Tag form fixture factory
 */
class TagFormFixtureFactory extends BaseFormFixtureFactory<TagFormData, TagCreateInput> {
    constructor() {
        super({
            scenarios: {
                minimal: {
                    tag: "javascript"
                },
                complete: {
                    tag: "typescript",
                    description: "TypeScript programming language",
                    language: "en",
                    isPopular: true,
                    suggestedCategory: "programming"
                },
                invalid: {
                    tag: "" // Empty tag
                },
                // Domain-specific scenarios
                programming: {
                    tag: "react",
                    description: "React JavaScript library",
                    suggestedCategory: "frontend"
                },
                aiml: {
                    tag: "machine-learning",
                    description: "Machine learning and AI",
                    suggestedCategory: "ai"
                },
                workflow: {
                    tag: "automation",
                    description: "Workflow automation",
                    suggestedCategory: "productivity"
                }
            },
            
            validate: createValidationAdapter<TagFormData>(
                async (data) => {
                    // UI-specific validation
                    const errors: string[] = [];
                    
                    if (data.tag && data.tag.includes(" ")) {
                        errors.push("tag: Tags cannot contain spaces");
                    }
                    
                    if (data.tag && data.tag.length > 50) {
                        errors.push("tag: Tag too long (max 50 characters)");
                    }
                    
                    if (errors.length > 0) {
                        return { isValid: false, errors };
                    }
                    
                    // Use shared validation
                    const apiInput = this.shapeToAPI!(data);
                    return tagValidation.create.validate(apiInput);
                }
            ),
            
            shapeToAPI: (formData) => {
                // Build translations if description provided
                const translations = formData.description ? [{
                    id: generatePK().toString(),
                    language: formData.language || "en",
                    description: formData.description
                }] : [];
                
                // Use the shape function from shared
                return shapeTag.create({
                    __typename: "Tag",
                    id: generatePK().toString(),
                    tag: formData.tag.toLowerCase(), // Normalize to lowercase
                    translations
                });
            }
        });
    }
    
    /**
     * Create multiple tags at once (common use case)
     */
    createMultipleTags(tags: string[]): TagFormData[] {
        return tags.map(tag => ({
            tag: tag.toLowerCase()
        }));
    }
    
    /**
     * Create tag with translation
     */
    createWithTranslation(
        tag: string,
        description: string,
        language: string = "en"
    ): TagFormData {
        return {
            tag,
            description,
            language
        };
    }
}

/**
 * Tag MSW handler factory
 */
class TagMSWHandlerFactory extends BaseMSWHandlerFactory<TagCreateInput, TagUpdateInput, Tag> {
    constructor() {
        super({
            baseUrl: "/api",
            endpoints: {
                create: "/tag",
                update: "/tag",
                delete: "/tag",
                find: "/tag",
                list: "/tags"
            },
            successResponses: {
                create: (input) => ({
                    __typename: "Tag" as const,
                    id: input.id || generatePK().toString(),
                    created_at: new Date().toISOString(),
                    tag: input.tag,
                    translations: input.translationsCreate || [],
                    // Populate with reasonable defaults
                    bookmarks: 0,
                    you: {
                        isBookmarked: false
                    }
                }),
                update: (input) => ({
                    ...sharedTagFixtures.complete.find,
                    ...input,
                    translations: input.translationsUpdate || []
                }),
                find: (id) => ({
                    ...sharedTagFixtures.complete.find,
                    id
                }),
                list: () => [
                    sharedTagFixtures.minimal.find,
                    sharedTagFixtures.complete.find,
                    {
                        ...sharedTagFixtures.minimal.find,
                        id: generatePK().toString(),
                        tag: "popular",
                        bookmarks: 100
                    }
                ]
            },
            validate: {
                create: (input) => {
                    const errors: string[] = [];
                    
                    if (!input.tag || input.tag.trim().length === 0) {
                        errors.push("Tag is required");
                    }
                    
                    if (input.tag && input.tag.includes(" ")) {
                        errors.push("Tags cannot contain spaces");
                    }
                    
                    return {
                        isValid: errors.length === 0,
                        errors
                    };
                }
            }
        });
    }
    
    /**
     * Create handlers for tag suggestions endpoint
     */
    createSuggestionHandlers(suggestions: string[]) {
        return this.createCustomHandler({
            method: "GET",
            path: "/tags/suggestions",
            response: suggestions.map(tag => ({
                __typename: "Tag" as const,
                id: generatePK().toString(),
                tag,
                created_at: new Date().toISOString(),
                bookmarks: Math.floor(Math.random() * 100)
            }))
        });
    }
}

/**
 * Tag UI state fixture factory
 */
class TagUIStateFixtureFactory implements UIStateFixtureFactory<TagUIState> {
    createLoadingState(): TagUIState {
        return {
            isLoading: true,
            tags: [],
            selectedTags: [],
            suggestions: [],
            error: null
        };
    }
    
    createErrorState(error: { message: string }): TagUIState {
        return {
            isLoading: false,
            tags: [],
            selectedTags: [],
            suggestions: [],
            error: error.message
        };
    }
    
    createSuccessState(data: Tag[]): TagUIState {
        return {
            isLoading: false,
            tags: data,
            selectedTags: [],
            suggestions: [],
            error: null
        };
    }
    
    createEmptyState(): TagUIState {
        return {
            isLoading: false,
            tags: [],
            selectedTags: [],
            suggestions: [],
            error: null
        };
    }
    
    transitionToLoading(currentState: TagUIState): TagUIState {
        return {
            ...currentState,
            isLoading: true,
            error: null
        };
    }
    
    transitionToSuccess(currentState: TagUIState, data: Tag[]): TagUIState {
        return {
            ...currentState,
            isLoading: false,
            tags: data,
            error: null
        };
    }
    
    transitionToError(currentState: TagUIState, error: { message: string }): TagUIState {
        return {
            ...currentState,
            isLoading: false,
            error: error.message
        };
    }
    
    /**
     * Tag-specific state transitions
     */
    selectTag(currentState: TagUIState, tagId: string): TagUIState {
        return {
            ...currentState,
            selectedTags: [...currentState.selectedTags, tagId]
        };
    }
    
    deselectTag(currentState: TagUIState, tagId: string): TagUIState {
        return {
            ...currentState,
            selectedTags: currentState.selectedTags.filter(id => id !== tagId)
        };
    }
    
    setSuggestions(currentState: TagUIState, suggestions: Tag[]): TagUIState {
        return {
            ...currentState,
            suggestions
        };
    }
}

/**
 * Complete Tag fixture factory
 */
export class TagFixtureFactory implements UIFixtureFactory<
    TagFormData,
    TagCreateInput,
    TagUpdateInput,
    Tag,
    TagUIState
> {
    readonly objectType = "tag";
    
    form: TagFormFixtureFactory;
    roundTrip: RoundTripOrchestrator<TagFormData, Tag>;
    handlers: TagMSWHandlerFactory;
    states: TagUIStateFixtureFactory;
    componentUtils: ComponentTestUtils<any>;
    
    constructor(apiClient: TestAPIClient, dbVerifier: DatabaseVerifier) {
        this.form = new TagFormFixtureFactory();
        this.handlers = new TagMSWHandlerFactory();
        this.states = new TagUIStateFixtureFactory();
        
        // Initialize round-trip orchestrator
        this.roundTrip = new BaseRoundTripOrchestrator({
            apiClient,
            dbVerifier,
            formFixture: this.form,
            endpoints: {
                create: "/api/tag",
                update: "/api/tag",
                delete: "/api/tag",
                find: "/api/tag"
            },
            tableName: "tag",
            fieldMappings: {
                tag: "tag"
            }
        });
        
        // Component utils would be initialized here
        this.componentUtils = {} as any; // Placeholder
    }
    
    createFormData(scenario: "minimal" | "complete" | string = "minimal"): TagFormData {
        return this.form.createFormData(scenario);
    }
    
    createAPIInput(formData: TagFormData): TagCreateInput {
        return this.form.transformToAPIInput(formData);
    }
    
    createMockResponse(overrides?: Partial<Tag>): Tag {
        return {
            ...sharedTagFixtures.complete.find,
            ...overrides
        };
    }
    
    setupMSW(scenario: "success" | "error" | "loading" = "success"): void {
        // This would integrate with the MSW server
        console.log(`Setting up MSW handlers for scenario: ${scenario}`);
    }
    
    async testCreateFlow(formData?: TagFormData): Promise<Tag> {
        const data = formData || this.createFormData("minimal");
        const result = await this.roundTrip.testCreateFlow(data);
        
        if (!result.success) {
            throw new Error(`Create flow failed: ${result.errors?.join(", ")}`);
        }
        
        return result.metadata?.id as unknown as Tag;
    }
    
    async testUpdateFlow(id: string, updates: Partial<TagFormData>): Promise<Tag> {
        const result = await this.roundTrip.testUpdateFlow(id, updates);
        
        if (!result.success) {
            throw new Error(`Update flow failed: ${result.errors?.join(", ")}`);
        }
        
        return result.metadata?.updatedData as Tag;
    }
    
    async testDeleteFlow(id: string): Promise<boolean> {
        const result = await this.roundTrip.testDeleteFlow(id);
        return result.success;
    }
    
    async testRoundTrip(formData?: TagFormData) {
        const data = formData || this.createFormData("complete");
        const result = await this.roundTrip.executeFullCycle({
            formData: data,
            validateEachStep: true
        });
        
        if (!result.success) {
            throw new Error(`Round trip failed: ${result.errors?.join(", ")}`);
        }
        
        return {
            success: result.success,
            formData: data,
            apiResponse: result.data!.apiResponse,
            uiState: this.states.createSuccessState([result.data!.apiResponse])
        };
    }
    
    /**
     * Tag-specific test helpers
     */
    async testBulkCreate(tags: string[]): Promise<Tag[]> {
        const formDataArray = this.form.createMultipleTags(tags);
        const results: Tag[] = [];
        
        for (const formData of formDataArray) {
            const tag = await this.testCreateFlow(formData);
            results.push(tag);
        }
        
        return results;
    }
}

// Register in the global registry
// This would normally be done after creating the API client and DB verifier
// registerFixture("tag", new TagFixtureFactory(apiClient, dbVerifier));
/* c8 ignore stop */