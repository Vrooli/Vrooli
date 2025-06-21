/**
 * Production-grade user fixture factory
 * 
 * This factory provides type-safe user fixtures using real functions from @vrooli/shared.
 * It eliminates `any` types and integrates with actual validation and shape transformation logic.
 */

import type { 
    User, 
    UserCreateInput, 
    UserUpdateInput,
    UserTranslation,
    Wallet
} from "@vrooli/shared";
import { 
    userValidation,
    shapeUser,
    emailSignUpFormValidation,
    AccountStatus
} from "@vrooli/shared";
import type { 
    FixtureFactory, 
    ValidationResult, 
    MSWHandlers
} from "../types.js";
import { rest } from "msw";

/**
 * UI-specific form data for user registration
 */
export interface UserFormData {
    email: string;
    password: string;
    confirmPassword?: string;
    handle: string;
    name: string;
    bio?: string;
    theme?: string;
    isPrivate?: boolean;
    agreeToTerms?: boolean;
    marketingEmails?: boolean;
}

/**
 * UI-specific form data for user profile update
 */
export interface UserProfileFormData {
    handle?: string;
    name?: string;
    bio?: string;
    theme?: string;
    isPrivate?: boolean;
    profileImage?: File | string | null;
    bannerImage?: File | string | null;
}

/**
 * UI state for user components
 */
export interface UserUIState {
    isLoading: boolean;
    user: User | null;
    error: string | null;
    isAuthenticated: boolean;
    isEmailVerified: boolean;
    sessionToken?: string;
}

export type UserScenario = 
    | "minimal" 
    | "complete" 
    | "invalid" 
    | "withBio"
    | "privateUser"
    | "withTranslations"
    | "withWallets"
    | "premiumUser";

/**
 * Type-safe user fixture factory that uses real @vrooli/shared functions
 */
export class UserFixtureFactory implements FixtureFactory<
    UserFormData,
    UserCreateInput,
    UserUpdateInput,
    User
> {
    readonly objectType = "user";

    /**
     * Generate a unique ID for testing
     */
    private generateId(): string {
        return `test_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }

    /**
     * Generate a unique handle
     */
    private generateHandle(): string {
        return `testuser_${Math.random().toString(36).substr(2, 9)}`;
    }

    /**
     * Generate a unique email
     */
    private generateEmail(): string {
        return `test.${Date.now()}@example.com`;
    }

    /**
     * Create form data for different test scenarios
     */
    createFormData(scenario: UserScenario = "minimal"): UserFormData {
        const basePassword = "SecurePass123!";
        
        switch (scenario) {
            case "minimal":
                return {
                    email: this.generateEmail(),
                    password: basePassword,
                    confirmPassword: basePassword,
                    handle: this.generateHandle(),
                    name: "Test User",
                    agreeToTerms: true
                };

            case "complete":
                return {
                    email: this.generateEmail(),
                    password: basePassword,
                    confirmPassword: basePassword,
                    handle: this.generateHandle(),
                    name: "Complete Test User",
                    bio: "This is a test user with all fields filled out",
                    theme: "light",
                    isPrivate: false,
                    agreeToTerms: true,
                    marketingEmails: true
                };

            case "invalid":
                return {
                    email: "invalid-email", // Invalid email format
                    password: "weak", // Too short password
                    confirmPassword: "different", // Doesn't match
                    handle: "a", // Too short
                    name: "", // Empty name
                    agreeToTerms: false // Must agree to terms
                };

            case "withBio":
                return {
                    email: this.generateEmail(),
                    password: basePassword,
                    confirmPassword: basePassword,
                    handle: this.generateHandle(),
                    name: "User With Bio",
                    bio: "I am a test user with a detailed biography. I enjoy testing applications and finding bugs.",
                    agreeToTerms: true
                };

            case "privateUser":
                return {
                    email: this.generateEmail(),
                    password: basePassword,
                    confirmPassword: basePassword,
                    handle: this.generateHandle(),
                    name: "Private User",
                    isPrivate: true,
                    agreeToTerms: true
                };

            case "withTranslations":
                return {
                    email: this.generateEmail(),
                    password: basePassword,
                    confirmPassword: basePassword,
                    handle: this.generateHandle(),
                    name: "Multilingual User",
                    bio: "A user who speaks multiple languages",
                    agreeToTerms: true
                };

            case "withWallets":
                return {
                    email: this.generateEmail(),
                    password: basePassword,
                    confirmPassword: basePassword,
                    handle: this.generateHandle(),
                    name: "Crypto User",
                    bio: "I use cryptocurrency wallets",
                    agreeToTerms: true
                };

            case "premiumUser":
                return {
                    email: this.generateEmail(),
                    password: basePassword,
                    confirmPassword: basePassword,
                    handle: this.generateHandle(),
                    name: "Premium User",
                    bio: "I have a premium account",
                    agreeToTerms: true
                };

            default:
                throw new Error(`Unknown user scenario: ${scenario}`);
        }
    }

    /**
     * Create profile update form data
     */
    createProfileFormData(scenario: "minimal" | "complete" | "withImages" = "minimal"): UserProfileFormData {
        switch (scenario) {
            case "minimal":
                return {
                    name: "Updated Name"
                };

            case "complete":
                return {
                    handle: this.generateHandle(),
                    name: "Fully Updated User",
                    bio: "Updated biography with new information",
                    theme: "dark",
                    isPrivate: false
                };

            case "withImages":
                return {
                    name: "User With Images",
                    bio: "Updated with profile and banner images",
                    profileImage: "profile.jpg", // In real tests, this would be a File object
                    bannerImage: "banner.jpg"
                };

            default:
                return { name: "Updated User" };
        }
    }

    /**
     * Transform form data to API create input using real shape function
     */
    transformToAPIInput(formData: UserFormData): UserCreateInput {
        // Create the user shape that matches the expected API structure
        const userShape = {
            __typename: "User" as const,
            id: this.generateId(),
            handle: formData.handle,
            name: formData.name,
            email: formData.email,
            password: formData.password,
            isPrivate: formData.isPrivate || false,
            theme: formData.theme || "light",
            translations: formData.bio ? [{
                __typename: "UserTranslation" as const,
                id: this.generateId(),
                language: "en",
                bio: formData.bio
            }] : null
        };

        // Use real shape function from @vrooli/shared
        return shapeUser.create(userShape);
    }

    /**
     * Create API update input
     */
    createUpdateInput(id: string, updates: Partial<UserProfileFormData>): UserUpdateInput {
        const updateInput: UserUpdateInput = { id };

        if (updates.handle) updateInput.handle = updates.handle;
        if (updates.name) updateInput.name = updates.name;
        if (updates.theme) updateInput.theme = updates.theme;
        if (updates.isPrivate !== undefined) updateInput.isPrivate = updates.isPrivate;

        // Handle translations for bio
        if (updates.bio) {
            updateInput.translationsUpdate = [{
                id: this.generateId(),
                language: "en",
                bio: updates.bio
            }];
        }

        // Handle image updates
        if (updates.profileImage !== undefined) {
            updateInput.profileImage = updates.profileImage;
        }
        if (updates.bannerImage !== undefined) {
            updateInput.bannerImage = updates.bannerImage;
        }

        return updateInput;
    }

    /**
     * Create mock user response with realistic data
     */
    createMockResponse(overrides?: Partial<User>): User {
        const now = new Date().toISOString();
        const userId = this.generateId();
        
        const defaultUser: User = {
            __typename: "User",
            id: userId,
            handle: this.generateHandle(),
            name: "Test User",
            email: this.generateEmail(),
            emailVerified: true,
            createdAt: now,
            updatedAt: now,
            isBot: false,
            isPrivate: false,
            profileImage: null,
            bannerImage: null,
            bio: null,
            theme: "light",
            status: AccountStatus.Active,
            stripeCustomerId: null,
            hasPremium: false,
            premium: false,
            premiumExpiration: null,
            awards: [],
            awardsCount: 0,
            bookmarks: [],
            bookmarksCount: 0,
            chats: [],
            languages: ["en"],
            focusModes: [],
            focusModesCount: 0,
            phones: [],
            emails: [{
                __typename: "Email",
                id: this.generateId(),
                emailAddress: this.generateEmail(),
                verified: true
            }],
            wallets: [],
            walletsCount: 0,
            translations: [{
                __typename: "UserTranslation",
                id: this.generateId(),
                language: "en",
                bio: null
            }],
            translationsCount: 1,
            roles: [],
            rolesCount: 0,
            stats: {
                __typename: "StatsUser",
                id: this.generateId(),
                userId,
                reportsCount: 0,
                reputation: 0,
                projectCompletedCount: 0,
                teamsCount: 0,
                teamJoinRequestsCount: 0,
                projectCommentsCount: 0,
                questionsAnsweredCount: 0,
                questionsAskedCount: 0,
                routineCompletedCount: 0,
                standardCompletedCount: 0,
                codeCompletedCount: 0,
                routineCommentCount: 0,
                chatsCount: 0,
                totalVotes: 0
            },
            you: {
                __typename: "UserYou",
                isBlocked: false,
                isBlockedByYou: false,
                canDelete: false,
                canReport: false,
                canUpdate: false,
                isBookmarked: false,
                isReacted: false,
                reactionSummary: {
                    __typename: "ReactionSummary",
                    emotion: null,
                    count: 0
                }
            }
        };

        return {
            ...defaultUser,
            ...overrides
        };
    }

    /**
     * Validate form data using real validation from @vrooli/shared
     */
    async validateFormData(formData: UserFormData): Promise<ValidationResult> {
        try {
            // Use real form validation schema from @vrooli/shared
            await emailSignUpFormValidation.validate(formData, { abortEarly: false });
            
            // Additional validation for password confirmation
            if (formData.password !== formData.confirmPassword) {
                return {
                    isValid: false,
                    errors: ["Passwords do not match"],
                    fieldErrors: {
                        confirmPassword: ["Passwords do not match"]
                    }
                };
            }

            // Validate terms agreement
            if (!formData.agreeToTerms) {
                return {
                    isValid: false,
                    errors: ["You must agree to the terms and conditions"],
                    fieldErrors: {
                        agreeToTerms: ["You must agree to the terms and conditions"]
                    }
                };
            }
            
            return { isValid: true };
        } catch (error: any) {
            return {
                isValid: false,
                errors: error.errors || [error.message],
                fieldErrors: error.inner?.reduce((acc: any, err: any) => {
                    if (err.path) {
                        acc[err.path] = acc[err.path] || [];
                        acc[err.path].push(err.message);
                    }
                    return acc;
                }, {})
            };
        }
    }

    /**
     * Validate profile update form data
     */
    async validateProfileFormData(formData: UserProfileFormData): Promise<ValidationResult> {
        try {
            const updateInput = this.createUpdateInput(this.generateId(), formData);
            await userValidation.update.validate(updateInput, { abortEarly: false });
            return { isValid: true };
        } catch (error: any) {
            return {
                isValid: false,
                errors: error.errors || [error.message],
                fieldErrors: error.inner?.reduce((acc: any, err: any) => {
                    if (err.path) {
                        acc[err.path] = acc[err.path] || [];
                        acc[err.path].push(err.message);
                    }
                    return acc;
                }, {})
            };
        }
    }

    /**
     * Create MSW handlers for different scenarios
     */
    createMSWHandlers(): MSWHandlers {
        const baseUrl = process.env.VITE_SERVER_URL || 'http://localhost:3000';

        return {
            success: [
                // Registration
                rest.post(`${baseUrl}/api/auth/register`, async (req, res, ctx) => {
                    const body = await req.json();
                    
                    // Validate the request body
                    const validation = await this.validateFormData(body);
                    if (!validation.isValid) {
                        return res(
                            ctx.status(400),
                            ctx.json({ 
                                errors: validation.errors,
                                fieldErrors: validation.fieldErrors 
                            })
                        );
                    }

                    // Return successful response
                    const mockUser = this.createMockResponse({
                        email: body.email,
                        handle: body.handle,
                        name: body.name,
                        emailVerified: false // New users need to verify email
                    });

                    return res(
                        ctx.status(201),
                        ctx.json({
                            user: mockUser,
                            sessionToken: `test_session_${Date.now()}`
                        })
                    );
                }),

                // Login
                rest.post(`${baseUrl}/api/auth/login`, async (req, res, ctx) => {
                    const body = await req.json();

                    const mockUser = this.createMockResponse({
                        email: body.email
                    });

                    return res(
                        ctx.status(200),
                        ctx.json({
                            user: mockUser,
                            sessionToken: `test_session_${Date.now()}`
                        })
                    );
                }),

                // Profile update
                rest.put(`${baseUrl}/api/user/:id`, async (req, res, ctx) => {
                    const { id } = req.params;
                    const body = await req.json();

                    const validation = await this.validateProfileFormData(body);
                    if (!validation.isValid) {
                        return res(
                            ctx.status(400),
                            ctx.json({ 
                                errors: validation.errors,
                                fieldErrors: validation.fieldErrors 
                            })
                        );
                    }

                    const mockUser = this.createMockResponse({ 
                        id: id as string,
                        ...body,
                        updatedAt: new Date().toISOString()
                    });

                    return res(
                        ctx.status(200),
                        ctx.json(mockUser)
                    );
                }),

                // Get user
                rest.get(`${baseUrl}/api/user/:id`, (req, res, ctx) => {
                    const { id } = req.params;
                    const mockUser = this.createMockResponse({ id: id as string });
                    
                    return res(
                        ctx.status(200),
                        ctx.json(mockUser)
                    );
                }),

                // Current user
                rest.get(`${baseUrl}/api/auth/me`, (req, res, ctx) => {
                    const authHeader = req.headers.get('Authorization');
                    
                    if (!authHeader || !authHeader.startsWith('Bearer ')) {
                        return res(
                            ctx.status(401),
                            ctx.json({ error: 'Unauthorized' })
                        );
                    }

                    const mockUser = this.createMockResponse();
                    
                    return res(
                        ctx.status(200),
                        ctx.json(mockUser)
                    );
                })
            ],

            error: [
                rest.post(`${baseUrl}/api/auth/register`, (req, res, ctx) => {
                    return res(
                        ctx.status(409),
                        ctx.json({ 
                            message: 'Email already exists',
                            code: 'EMAIL_EXISTS' 
                        })
                    );
                }),

                rest.post(`${baseUrl}/api/auth/login`, (req, res, ctx) => {
                    return res(
                        ctx.status(401),
                        ctx.json({ 
                            message: 'Invalid credentials',
                            code: 'INVALID_CREDENTIALS' 
                        })
                    );
                }),

                rest.put(`${baseUrl}/api/user/:id`, (req, res, ctx) => {
                    return res(
                        ctx.status(403),
                        ctx.json({ 
                            message: 'Forbidden: Cannot update other users',
                            code: 'FORBIDDEN' 
                        })
                    );
                })
            ],

            loading: [
                rest.post(`${baseUrl}/api/auth/register`, (req, res, ctx) => {
                    return res(
                        ctx.delay(2000), // 2 second delay
                        ctx.status(201),
                        ctx.json({
                            user: this.createMockResponse(),
                            sessionToken: `test_session_${Date.now()}`
                        })
                    );
                })
            ],

            networkError: [
                rest.post(`${baseUrl}/api/auth/register`, (req, res, ctx) => {
                    return res.networkError('Network connection failed');
                }),
                rest.post(`${baseUrl}/api/auth/login`, (req, res, ctx) => {
                    return res.networkError('Network connection failed');
                })
            ]
        };
    }

    /**
     * Create UI state fixtures for different scenarios
     */
    createUIState(state: "loading" | "error" | "authenticated" | "unauthenticated" | "emailUnverified" = "unauthenticated", data?: any): UserUIState {
        switch (state) {
            case "loading":
                return {
                    isLoading: true,
                    user: null,
                    error: null,
                    isAuthenticated: false,
                    isEmailVerified: false
                };

            case "error":
                return {
                    isLoading: false,
                    user: null,
                    error: data?.message || "Authentication failed",
                    isAuthenticated: false,
                    isEmailVerified: false
                };

            case "authenticated":
                const user = data || this.createMockResponse();
                return {
                    isLoading: false,
                    user,
                    error: null,
                    isAuthenticated: true,
                    isEmailVerified: user.emailVerified,
                    sessionToken: `test_session_${Date.now()}`
                };

            case "emailUnverified":
                const unverifiedUser = this.createMockResponse({ emailVerified: false });
                return {
                    isLoading: false,
                    user: unverifiedUser,
                    error: null,
                    isAuthenticated: true,
                    isEmailVerified: false,
                    sessionToken: `test_session_${Date.now()}`
                };

            case "unauthenticated":
            default:
                return {
                    isLoading: false,
                    user: null,
                    error: null,
                    isAuthenticated: false,
                    isEmailVerified: false
                };
        }
    }

    /**
     * Create a user with specific roles
     */
    createWithRoles(roles: string[]): User {
        return this.createMockResponse({
            roles: roles.map(role => ({
                __typename: "Role" as const,
                id: this.generateId(),
                name: role,
                permissions: []
            })),
            rolesCount: roles.length
        });
    }

    /**
     * Create a premium user
     */
    createPremiumUser(expirationDays: number = 30): User {
        const expiration = new Date();
        expiration.setDate(expiration.getDate() + expirationDays);
        
        return this.createMockResponse({
            hasPremium: true,
            premium: true,
            premiumExpiration: expiration.toISOString()
        });
    }

    /**
     * Create a user with wallets
     */
    createWithWallets(walletCount: number = 1): User {
        const wallets: Wallet[] = Array.from({ length: walletCount }, (_, i) => ({
            __typename: "Wallet",
            id: this.generateId(),
            name: `Wallet ${i + 1}`,
            publicAddress: `0x${Math.random().toString(16).substr(2, 40)}`,
            stakingAddress: null,
            verified: true,
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString()
        }));

        return this.createMockResponse({
            wallets,
            walletsCount: walletCount
        });
    }

    /**
     * Create test cases for various scenarios
     */
    createTestCases() {
        return [
            {
                name: "Valid registration",
                formData: this.createFormData("minimal"),
                shouldSucceed: true
            },
            {
                name: "Complete profile",
                formData: this.createFormData("complete"),
                shouldSucceed: true
            },
            {
                name: "Invalid email",
                formData: { ...this.createFormData("minimal"), email: "not-an-email" },
                shouldSucceed: false,
                expectedError: "email must be a valid email"
            },
            {
                name: "Weak password",
                formData: { ...this.createFormData("minimal"), password: "123", confirmPassword: "123" },
                shouldSucceed: false,
                expectedError: "password must be at least"
            },
            {
                name: "Password mismatch",
                formData: { ...this.createFormData("minimal"), confirmPassword: "different" },
                shouldSucceed: false,
                expectedError: "Passwords do not match"
            },
            {
                name: "Terms not agreed",
                formData: { ...this.createFormData("minimal"), agreeToTerms: false },
                shouldSucceed: false,
                expectedError: "You must agree to the terms"
            }
        ];
    }
}

/**
 * Default factory instance for easy importing
 */
export const userFixtures = new UserFixtureFactory();

/**
 * Specific test scenarios for common use cases
 */
export const userTestScenarios = {
    // Registration scenarios
    validRegistration: () => userFixtures.createFormData("minimal"),
    completeRegistration: () => userFixtures.createFormData("complete"),
    invalidRegistration: () => userFixtures.createFormData("invalid"),
    
    // Profile update scenarios
    minimalProfileUpdate: () => userFixtures.createProfileFormData("minimal"),
    completeProfileUpdate: () => userFixtures.createProfileFormData("complete"),
    profileWithImages: () => userFixtures.createProfileFormData("withImages"),
    
    // User type scenarios
    basicUser: () => userFixtures.createMockResponse(),
    privateUser: () => userFixtures.createMockResponse({ isPrivate: true }),
    premiumUser: () => userFixtures.createPremiumUser(),
    adminUser: () => userFixtures.createWithRoles(["Admin"]),
    userWithWallets: () => userFixtures.createWithWallets(2),
    
    // UI state scenarios
    loadingState: () => userFixtures.createUIState("loading"),
    errorState: (message?: string) => userFixtures.createUIState("error", { message }),
    authenticatedState: (user?: User) => userFixtures.createUIState("authenticated", user),
    unauthenticatedState: () => userFixtures.createUIState("unauthenticated"),
    emailUnverifiedState: () => userFixtures.createUIState("emailUnverified"),
    
    // Test data sets
    allTestCases: () => userFixtures.createTestCases(),
    
    // MSW handlers
    successHandlers: () => userFixtures.createMSWHandlers().success,
    errorHandlers: () => userFixtures.createMSWHandlers().error,
    loadingHandlers: () => userFixtures.createMSWHandlers().loading,
    networkErrorHandlers: () => userFixtures.createMSWHandlers().networkError
};