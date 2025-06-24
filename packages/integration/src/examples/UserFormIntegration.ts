/**
 * User Form Integration Testing Example
 * 
 * This demonstrates comprehensive round-trip testing for User operations
 * including signup, profile updates, and authentication flows using the
 * IntegrationFormTestFactory.
 */

import { 
    type User, 
    type EmailSignUpInput, 
    type ProfileUpdateInput, 
    type UserShape,
    emailSignUpValidation,
    profileUpdateValidation,
    shapeUser,
    endpointsAuth,
    endpointsUser,
    DUMMY_ID,
} from "@vrooli/shared";
import { createIntegrationFormTestFactory } from "../engine/IntegrationFormTestFactory.js";
import { getPrisma } from "../../setup/test-setup.js";

/**
 * Form data types for user testing (simulates UI form data)
 */
interface SignupFormData {
    name: string;
    email: string;
    password: string;
    confirmPassword: string;
    marketingEmails: boolean;
    theme?: string;
    agreeToTerms: boolean;
}

interface ProfileUpdateFormData {
    name?: string;
    bio?: string;
    handle?: string;
    theme?: string;
    language?: string;
    // Profile image and banner
    profileImage?: string;
    bannerImage?: string;
    // Privacy settings
    isPrivate?: boolean;
    // Notification preferences
    emailNotifications?: boolean;
    pushNotifications?: boolean;
}

/**
 * Test fixtures for different user scenarios
 */
export const userSignupFormFixtures: Record<string, SignupFormData> = {
    minimal: {
        name: "Test User",
        email: "test@example.com",
        password: "SecurePassword123!",
        confirmPassword: "SecurePassword123!",
        marketingEmails: false,
        agreeToTerms: true,
    },
    
    complete: {
        name: "Complete Test User",
        email: "complete@example.com",
        password: "VerySecurePassword123!",
        confirmPassword: "VerySecurePassword123!",
        marketingEmails: true,
        theme: "dark",
        agreeToTerms: true,
    },
    
    weakPassword: {
        name: "Test User",
        email: "weak@example.com",
        password: "123",
        confirmPassword: "123",
        marketingEmails: false,
        agreeToTerms: true,
    },
    
    invalidEmail: {
        name: "Test User",
        email: "invalid-email",
        password: "SecurePassword123!",
        confirmPassword: "SecurePassword123!",
        marketingEmails: false,
        agreeToTerms: true,
    },
    
    passwordMismatch: {
        name: "Test User",
        email: "mismatch@example.com",
        password: "SecurePassword123!",
        confirmPassword: "DifferentPassword123!",
        marketingEmails: false,
        agreeToTerms: true,
    },
    
    noTermsAgreement: {
        name: "Test User",
        email: "noterms@example.com",
        password: "SecurePassword123!",
        confirmPassword: "SecurePassword123!",
        marketingEmails: false,
        agreeToTerms: false,
    },
};

export const userProfileFormFixtures: Record<string, ProfileUpdateFormData> = {
    minimal: {
        bio: "Updated bio text",
    },
    
    complete: {
        name: "Updated Name",
        bio: "I am a developer interested in AI and automation. I love building things that help people be more productive.",
        handle: "updated_handle",
        theme: "dark",
        language: "es",
        isPrivate: false,
        emailNotifications: true,
        pushNotifications: false,
    },
    
    privacyUpdate: {
        isPrivate: true,
        emailNotifications: false,
        pushNotifications: false,
    },
    
    invalidHandle: {
        handle: "", // Empty handle should fail validation
    },
    
    longBio: {
        bio: "A".repeat(5000), // Very long bio
        handle: "long_bio_user",
    },
};

/**
 * Convert signup form data to UserShape
 */
function signupFormToShape(formData: SignupFormData): UserShape {
    return {
        __typename: "User",
        id: DUMMY_ID,
        name: formData.name,
        emails: [{
            __typename: "Email",
            id: DUMMY_ID,
            emailAddress: formData.email,
            verified: false,
        }],
        theme: formData.theme || "light",
        marketingEmails: formData.marketingEmails,
    };
}

/**
 * Convert profile update form data to UserShape
 */
function profileUpdateFormToShape(formData: ProfileUpdateFormData): UserShape {
    const shape: UserShape = {
        __typename: "User",
        id: DUMMY_ID,
    };

    if (formData.name) shape.name = formData.name;
    if (formData.bio) shape.bio = formData.bio;
    if (formData.handle) shape.handle = formData.handle;
    if (formData.theme) shape.theme = formData.theme;
    if (formData.language) shape.language = formData.language;
    if (formData.isPrivate !== undefined) shape.isPrivate = formData.isPrivate;

    return shape;
}

/**
 * Transform user signup values for API calls
 */
function transformSignupValues(values: UserShape, existing: UserShape, isCreate: boolean): EmailSignUpInput | ProfileUpdateInput {
    if (isCreate) {
        // For signup, we need to extract email and password info
        return {
            name: values.name || "",
            email: values.emails?.[0]?.emailAddress || "",
            password: "SecurePassword123!", // This would come from form data in real scenario
            confirmPassword: "SecurePassword123!",
            marketingEmails: values.marketingEmails || false,
        } as EmailSignUpInput;
    } else {
        // For profile updates
        return shapeUser.update(existing, values) as ProfileUpdateInput;
    }
}

/**
 * Transform profile update values for API calls
 */
function transformProfileUpdateValues(values: UserShape, existing: UserShape, isCreate: boolean): ProfileUpdateInput {
    return shapeUser.update(existing, values) as ProfileUpdateInput;
}

/**
 * Find user in database
 */
async function findUserInDatabase(id: string): Promise<User | null> {
    const prisma = getPrisma();
    if (!prisma) return null;
    
    try {
        return await prisma.user.findUnique({
            where: { id },
            include: {
                emails: true,
                auths: true,
                memberships: {
                    include: {
                        team: true,
                    },
                },
                wallets: true,
            },
        });
    } catch (error) {
        console.error("Error finding user in database:", error);
        return null;
    }
}

/**
 * Integration test factory for User signup forms
 */
export const userSignupFormIntegrationFactory = createIntegrationFormTestFactory({
    objectType: "User",
    validation: {
        create: emailSignUpValidation,
        update: profileUpdateValidation,
    },
    transformFunction: transformSignupValues,
    endpoints: {
        create: endpointsAuth.emailSignUp,
        update: endpointsUser.profileUpdate,
    },
    formFixtures: userSignupFormFixtures,
    formToShape: signupFormToShape,
    findInDatabase: findUserInDatabase,
    prismaModel: "user",
});

/**
 * Integration test factory for User profile update forms
 */
export const userProfileFormIntegrationFactory = createIntegrationFormTestFactory({
    objectType: "User",
    validation: {
        create: profileUpdateValidation,
        update: profileUpdateValidation,
    },
    transformFunction: transformProfileUpdateValues,
    endpoints: {
        create: endpointsUser.profileUpdate,
        update: endpointsUser.profileUpdate,
    },
    formFixtures: userProfileFormFixtures,
    formToShape: profileUpdateFormToShape,
    findInDatabase: findUserInDatabase,
    prismaModel: "user",
});

/**
 * Test cases for user form integration
 */
export const userSignupIntegrationTestCases = userSignupFormIntegrationFactory.generateIntegrationTestCases();
export const userProfileIntegrationTestCases = userProfileFormIntegrationFactory.generateIntegrationTestCases();

/**
 * Helper function to create authenticated user session for testing
 */
export async function createAuthenticatedUserSession(userId: string): Promise<{ sessionToken: string; languages: string[] }> {
    const prisma = getPrisma();
    if (!prisma) throw new Error("Prisma not available");

    // Create a session token for the user
    const session = await prisma.session.create({
        data: {
            id: `test-session-${Date.now()}`,
            user: { connect: { id: userId } },
            token: `test-token-${Date.now()}`,
            ipAddress: "127.0.0.1",
            userAgent: "Test Agent",
            isLoggedIn: true,
            languages: ["en"],
        },
    });

    return {
        sessionToken: session.token,
        languages: session.languages,
    };
}

/**
 * Helper function to verify user authentication state
 */
export async function verifyUserAuthState(userId: string): Promise<{
    hasVerifiedEmail: boolean;
    hasPassword: boolean;
    sessionCount: number;
}> {
    const prisma = getPrisma();
    if (!prisma) throw new Error("Prisma not available");

    const user = await prisma.user.findUnique({
        where: { id: userId },
        include: {
            emails: true,
            auths: true,
            sessions: true,
        },
    });

    if (!user) throw new Error("User not found");

    return {
        hasVerifiedEmail: user.emails.some(email => email.verified),
        hasPassword: user.auths.some(auth => auth.type === "Password"),
        sessionCount: user.sessions.length,
    };
}