/* c8 ignore start */
/**
 * User fixture factory implementation
 * 
 * This demonstrates the complete pattern for implementing fixtures for a Vrooli object type
 * with full type safety and integration with @vrooli/shared.
 */

import {
    type User,
    type ProfileUpdateInput,
    type EmailSignUpInput,
    shapeUser,
    userValidation,
    generatePK,
    userFixtures as sharedUserFixtures,
    botConfigFixtures
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
import { registerFixture } from "./index.js";

/**
 * User registration form data type
 * 
 * This includes UI-specific fields like confirmPassword and acceptTerms
 * that don't exist in the API input type.
 */
export interface UserRegistrationFormData {
    email: string;
    password: string;
    confirmPassword: string;
    handle: string;
    name: string;
    bio?: string;
    theme?: string;
    language?: string;
    acceptTerms: boolean;
    marketingEmails?: boolean;
    referralCode?: string;
}

/**
 * User profile update form data type
 */
export interface UserProfileFormData {
    handle?: string;
    name?: string;
    bio?: string;
    theme?: string;
    language?: string;
    isPrivate?: boolean;
    isPrivateMemberships?: boolean;
    isPrivateBookmarks?: boolean;
}

/**
 * User UI state type
 */
export interface UserUIState {
    isLoading: boolean;
    user: User | null;
    error: string | null;
    isEditing: boolean;
    hasUnsavedChanges: boolean;
}

/**
 * User form fixture factory
 */
class UserFormFixtureFactory extends BaseFormFixtureFactory<UserRegistrationFormData, EmailSignUpInput> {
    constructor() {
        super({
            scenarios: {
                minimal: {
                    email: "test@example.com",
                    password: "SecurePass123!",
                    confirmPassword: "SecurePass123!",
                    handle: "testuser",
                    name: "Test User",
                    acceptTerms: true
                },
                complete: {
                    email: "power@example.com",
                    password: "SuperSecure123!@#",
                    confirmPassword: "SuperSecure123!@#",
                    handle: "poweruser",
                    name: "Power User",
                    bio: "Passionate developer interested in AI and automation",
                    theme: "dark",
                    language: "en",
                    acceptTerms: true,
                    marketingEmails: true,
                    referralCode: "FRIEND2024"
                },
                invalid: {
                    email: "invalid-email",
                    password: "weak",
                    confirmPassword: "different",
                    handle: "a", // Too short
                    name: "",
                    acceptTerms: false
                },
                bot: {
                    email: "bot@vrooli.com",
                    password: "BotPass123!",
                    confirmPassword: "BotPass123!",
                    handle: "testbot",
                    name: "Test Bot",
                    acceptTerms: true
                }
            },
            
            validate: createValidationAdapter<UserRegistrationFormData>(
                async (data) => {
                    // Additional UI validation
                    const errors: string[] = [];
                    
                    if (data.password !== data.confirmPassword) {
                        errors.push("confirmPassword: Passwords do not match");
                    }
                    
                    if (!data.acceptTerms) {
                        errors.push("acceptTerms: You must accept the terms");
                    }
                    
                    if (errors.length > 0) {
                        return { isValid: false, errors };
                    }
                    
                    // Use shared validation for the rest
                    const apiInput = this.shapeToAPI!(data);
                    return userValidation.create.validate(apiInput);
                }
            ),
            
            shapeToAPI: (formData) => {
                // Transform to EmailSignUpInput
                return {
                    email: formData.email,
                    password: formData.password,
                    handle: formData.handle,
                    name: formData.name,
                    ...(formData.bio && { bio: formData.bio }),
                    ...(formData.theme && { theme: formData.theme }),
                    ...(formData.language && { language: formData.language }),
                    ...(formData.marketingEmails !== undefined && { 
                        marketingEmails: formData.marketingEmails 
                    }),
                    ...(formData.referralCode && { referralCode: formData.referralCode })
                };
            }
        });
    }
    
    /**
     * Create profile update form data
     */
    createProfileFormData(scenario: "minimal" | "complete" = "minimal"): UserProfileFormData {
        if (scenario === "minimal") {
            return {
                name: "Updated Name"
            };
        }
        
        return {
            handle: "newhandle",
            name: "New Name",
            bio: "Updated bio with more information",
            theme: "light",
            language: "es",
            isPrivate: false,
            isPrivateMemberships: true,
            isPrivateBookmarks: false
        };
    }
}

/**
 * User MSW handler factory
 */
class UserMSWHandlerFactory extends BaseMSWHandlerFactory<EmailSignUpInput, ProfileUpdateInput, User> {
    constructor() {
        super({
            baseUrl: "/api",
            endpoints: {
                create: "/auth/signup",
                update: "/user",
                delete: "/user",
                find: "/user",
                list: "/users"
            },
            successResponses: {
                create: (input) => ({
                    ...sharedUserFixtures.complete.find,
                    id: generatePK().toString(),
                    email: input.email,
                    handle: input.handle,
                    name: input.name,
                    bio: input.bio || ""
                }),
                update: (input) => ({
                    ...sharedUserFixtures.complete.find,
                    ...input
                }),
                find: (id) => ({
                    ...sharedUserFixtures.complete.find,
                    id
                })
            },
            validate: {
                create: (input) => {
                    const errors: string[] = [];
                    
                    if (!input.email || !input.email.includes("@")) {
                        errors.push("Invalid email");
                    }
                    
                    if (!input.password || input.password.length < 8) {
                        errors.push("Password too short");
                    }
                    
                    if (!input.handle || input.handle.length < 3) {
                        errors.push("Handle too short");
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
     * Create bot user handlers
     */
    createBotHandlers() {
        return this.createCustomHandler({
            method: "POST",
            path: "/bot",
            response: {
                ...sharedUserFixtures.complete.find,
                isBot: true,
                botSettings: botConfigFixtures.complete
            }
        });
    }
}

/**
 * User UI state fixture factory
 */
class UserUIStateFixtureFactory implements UIStateFixtureFactory<UserUIState> {
    createLoadingState(context?: { type: string }): UserUIState {
        return {
            isLoading: true,
            user: null,
            error: null,
            isEditing: false,
            hasUnsavedChanges: false
        };
    }
    
    createErrorState(error: { message: string }): UserUIState {
        return {
            isLoading: false,
            user: null,
            error: error.message,
            isEditing: false,
            hasUnsavedChanges: false
        };
    }
    
    createSuccessState(data: User): UserUIState {
        return {
            isLoading: false,
            user: data,
            error: null,
            isEditing: false,
            hasUnsavedChanges: false
        };
    }
    
    createEmptyState(): UserUIState {
        return {
            isLoading: false,
            user: null,
            error: null,
            isEditing: false,
            hasUnsavedChanges: false
        };
    }
    
    transitionToLoading(currentState: UserUIState): UserUIState {
        return {
            ...currentState,
            isLoading: true,
            error: null
        };
    }
    
    transitionToSuccess(currentState: UserUIState, data: User): UserUIState {
        return {
            ...currentState,
            isLoading: false,
            user: data,
            error: null,
            hasUnsavedChanges: false
        };
    }
    
    transitionToError(currentState: UserUIState, error: { message: string }): UserUIState {
        return {
            ...currentState,
            isLoading: false,
            error: error.message
        };
    }
}

/**
 * Complete User fixture factory
 */
export class UserFixtureFactory implements UIFixtureFactory<
    UserRegistrationFormData,
    EmailSignUpInput,
    ProfileUpdateInput,
    User,
    UserUIState
> {
    readonly objectType = "user";
    
    form: UserFormFixtureFactory;
    roundTrip: RoundTripOrchestrator<UserRegistrationFormData, User>;
    handlers: UserMSWHandlerFactory;
    states: UserUIStateFixtureFactory;
    componentUtils: ComponentTestUtils<any>;
    
    constructor(apiClient: TestAPIClient, dbVerifier: DatabaseVerifier) {
        this.form = new UserFormFixtureFactory();
        this.handlers = new UserMSWHandlerFactory();
        this.states = new UserUIStateFixtureFactory();
        
        // Initialize round-trip orchestrator
        this.roundTrip = new BaseRoundTripOrchestrator({
            apiClient,
            dbVerifier,
            formFixture: this.form,
            endpoints: {
                create: "/api/auth/signup",
                update: "/api/user",
                delete: "/api/user",
                find: "/api/user"
            },
            tableName: "user",
            fieldMappings: {
                email: "email",
                handle: "handle",
                name: "name",
                bio: "bio"
            }
        });
        
        // Component utils would be initialized here
        this.componentUtils = {} as any; // Placeholder
    }
    
    createFormData(scenario: "minimal" | "complete" | string = "minimal"): UserRegistrationFormData {
        return this.form.createFormData(scenario);
    }
    
    createAPIInput(formData: UserRegistrationFormData): EmailSignUpInput {
        return this.form.transformToAPIInput(formData);
    }
    
    createMockResponse(overrides?: Partial<User>): User {
        return {
            ...sharedUserFixtures.complete.find,
            ...overrides
        };
    }
    
    setupMSW(scenario: "success" | "error" | "loading" = "success"): void {
        // This would integrate with the MSW server
        // For now, it's a placeholder
        console.log(`Setting up MSW handlers for scenario: ${scenario}`);
    }
    
    async testCreateFlow(formData?: UserRegistrationFormData): Promise<User> {
        const data = formData || this.createFormData("minimal");
        const result = await this.roundTrip.testCreateFlow(data);
        
        if (!result.success) {
            throw new Error(`Create flow failed: ${result.errors?.join(", ")}`);
        }
        
        return result.metadata?.id as unknown as User;
    }
    
    async testUpdateFlow(id: string, updates: Partial<UserRegistrationFormData>): Promise<User> {
        const result = await this.roundTrip.testUpdateFlow(id, updates);
        
        if (!result.success) {
            throw new Error(`Update flow failed: ${result.errors?.join(", ")}`);
        }
        
        return result.metadata?.updatedData as User;
    }
    
    async testDeleteFlow(id: string): Promise<boolean> {
        const result = await this.roundTrip.testDeleteFlow(id);
        return result.success;
    }
    
    async testRoundTrip(formData?: UserRegistrationFormData) {
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
            uiState: this.states.createSuccessState(result.data!.apiResponse)
        };
    }
}

// Register in the global registry
// This would normally be done after creating the API client and DB verifier
// registerFixture("user", new UserFixtureFactory(apiClient, dbVerifier));
/* c8 ignore stop */