/**
 * Comment Form Integration Testing Example
 * 
 * This demonstrates how to set up comprehensive round-trip testing for the Comment form
 * using the IntegrationFormTestFactory and leveraging existing fixture systems.
 * 
 * Key improvements in this example:
 * - Uses existing fixtures from @vrooli/shared and @vrooli/server
 * - Demonstrates permission-based testing with different user roles
 * - Shows integration with enhanced helpers for workflow testing
 * - Includes error scenario testing with realistic error fixtures
 */

import { 
    type Comment, 
    type CommentCreateInput, 
    type CommentUpdateInput, 
    type CommentShape,
    commentValidation,
    shapeComment,
    endpointsComment,
    DUMMY_ID,
} from "@vrooli/shared";
import { createIntegrationFormTestFactory } from "../engine/IntegrationFormTestFactory.js";
import { getPrisma } from "../../setup/test-setup.js";
import { 
    commentFixtures, 
    projectFixtures, 
    routineFixtures,
    userPersonas,
    enhancedTestUtils,
    EnhancedDataFactory,
    EnhancedSessionManager,
} from "../fixtures/index.js";

/**
 * Form data type for comment testing
 */
interface CommentFormData {
    text: string;
    commentedOnType: "Project" | "Routine" | "Standard";
    commentedOnId: string;
    translations?: Array<{
        language: string;
        text: string;
    }>;
}

/**
 * Enhanced test fixtures leveraging existing shared fixtures
 * These extend the base commentFixtures with form-specific data
 */
export const commentFormFixtures: Record<string, CommentFormData> = {
    // Leverage existing shared fixtures as base
    minimal: {
        text: commentFixtures.minimal.create.text || "This is a test comment",
        commentedOnType: "Project",
        commentedOnId: "test-project-id",
    },
    
    complete: {
        text: commentFixtures.complete.create.text || "This is a comprehensive test comment with detailed feedback and multiple points to discuss.",
        commentedOnType: "Routine",
        commentedOnId: "test-routine-id",
        translations: commentFixtures.complete.create.translations?.map(t => ({
            language: t.language,
            text: t.text,
        })) || [
            { language: "en", text: "This is a comprehensive test comment with detailed feedback." },
            { language: "es", text: "Este es un comentario de prueba integral con comentarios detallados." },
        ],
    },
    
    edgeCase: {
        text: "A".repeat(32768), // Maximum allowed length
        commentedOnType: "Standard",
        commentedOnId: "test-standard-id",
    },
    
    // Use shared invalid fixtures
    invalid: {
        text: "", // Empty text should fail validation
        commentedOnType: "Project",
        commentedOnId: "test-project-id",
    },
    
    longText: {
        text: "A".repeat(50000), // Exceeds maximum length
        commentedOnType: "Project", 
        commentedOnId: "test-project-id",
    },

    // Additional scenarios using shared fixture patterns
    withPermissions: {
        text: "Comment requiring specific permissions",
        commentedOnType: "Project",
        commentedOnId: "test-project-id",
    },

    multiUser: {
        text: "Comment in multi-user scenario",
        commentedOnType: "Routine",
        commentedOnId: "test-routine-id",
    },
};

/**
 * Convert form data to CommentShape
 */
function commentFormToShape(formData: CommentFormData): CommentShape {
    return {
        __typename: "Comment",
        id: DUMMY_ID,
        commentedOn: {
            __typename: formData.commentedOnType,
            id: formData.commentedOnId,
        },
        translations: formData.translations || [{
            __typename: "CommentTranslation",
            id: DUMMY_ID,
            language: "en",
            text: formData.text,
        }],
    };
}

/**
 * Transform comment values for API calls
 */
function transformCommentValues(values: CommentShape, existing: CommentShape, isCreate: boolean): CommentCreateInput | CommentUpdateInput {
    return isCreate ? shapeComment.create(values) : shapeComment.update(existing, values);
}

/**
 * Find comment in database
 */
async function findCommentInDatabase(id: string): Promise<Comment | null> {
    const prisma = getPrisma();
    if (!prisma) return null;
    
    try {
        return await prisma.comment.findUnique({
            where: { id },
            include: {
                translations: true,
                commentedOn: true,
                creator: true,
            },
        });
    } catch (error) {
        console.error("Error finding comment in database:", error);
        return null;
    }
}

/**
 * Integration test factory for Comment forms
 */
export const commentFormIntegrationFactory = createIntegrationFormTestFactory({
    objectType: "Comment",
    validation: commentValidation,
    transformFunction: transformCommentValues,
    endpoints: {
        create: endpointsComment.createOne,
        update: endpointsComment.updateOne,
    },
    formFixtures: commentFormFixtures,
    formToShape: commentFormToShape,
    findInDatabase: findCommentInDatabase,
    prismaModel: "comment",
});

/**
 * Test cases for comment form integration
 */
export const commentIntegrationTestCases = commentFormIntegrationFactory.generateIntegrationTestCases();

/**
 * Enhanced helper function using existing database fixtures
 */
export async function createTestCommentTarget(type: "Project" | "Routine" | "Standard"): Promise<string> {
    // Use enhanced data factory to leverage existing fixtures
    await EnhancedDataFactory.initialize();

    switch (type) {
        case "Project":
            const project = await EnhancedDataFactory.createTestData(
                "project",
                "minimal",
                { withRelations: true }
            );
            return project.id;

        case "Routine":
            const routine = await EnhancedDataFactory.createTestData(
                "routine", 
                "minimal",
                { withRelations: true }
            );
            return routine.id;

        case "Standard":
            // For now, create manually as standard might not have fixtures yet
            const prisma = getPrisma();
            if (!prisma) throw new Error("Prisma not available");

            // Use enhanced session manager to get a user
            const session = await EnhancedSessionManager.getSession('standard');
            
            const standard = await prisma.standard.create({
                data: {
                    id: `test-standard-${Date.now()}`,
                    isPrivate: false,
                    creator: { connect: { id: session.id } },
                    owner: { connect: { id: session.id } },
                    versions: {
                        create: {
                            id: `test-standard-version-${Date.now()}`,
                            versionLabel: "1.0.0",
                            isComplete: true,
                            isPrivate: false,
                            creator: { connect: { id: session.id } },
                            root: { connect: { id: session.id } },
                            translations: {
                                create: {
                                    language: "en",
                                    name: "Test Standard",
                                    description: "A test standard for comment integration testing",
                                },
                            },
                        },
                    },
                },
            });
            return standard.id;

        default:
            throw new Error(`Unsupported comment target type: ${type}`);
    }
}

/**
 * Enhanced comment integration factory with shared fixtures
 */
export const enhancedCommentFormIntegrationFactory = createIntegrationFormTestFactory({
    objectType: "Comment",
    validation: commentValidation,
    transformFunction: transformCommentValues,
    endpoints: {
        create: endpointsComment.createOne,
        update: endpointsComment.updateOne,
    },
    formFixtures: commentFormFixtures,
    formToShape: commentFormToShape,
    findInDatabase: findCommentInDatabase,
    prismaModel: "comment",
    // Enable shared fixture integration
    useSharedFixtures: true,
    userRole: 'standard',
    integrationOptions: {
        withDatabase: true,
        withPermissions: true,
        withEvents: false,
        withPerformance: true,
    },
});

/**
 * Advanced test scenarios using enhanced capabilities
 */
export const advancedCommentTestScenarios = {
    /**
     * Test comment creation with different user permissions
     */
    async testPermissionScenarios() {
        const results = [];
        
        for (const role of ['admin', 'standard', 'guest'] as const) {
            try {
                const result = await enhancedCommentFormIntegrationFactory.testWithSharedFixtures(
                    'minimal',
                    { userRole: role, withPermissions: true }
                );
                results.push({ role, success: result.success, result });
            } catch (error) {
                results.push({ 
                    role, 
                    success: false, 
                    error: error instanceof Error ? error.message : String(error) 
                });
            }
        }
        
        return results;
    },

    /**
     * Test comment workflow with multiple users
     */
    async testMultiUserWorkflow() {
        return await enhancedTestUtils.executeWorkflow(
            'contentWorkflow',
            'collaboration',
            {
                sessions: await EnhancedSessionManager.getMultipleSessions(['admin', 'standard']),
            }
        );
    },

    /**
     * Test error scenarios
     */
    async testErrorScenarios() {
        const errorTests = [];
        
        // Test API errors
        const apiErrorResult = await enhancedCommentFormIntegrationFactory.testErrorScenarios(
            'apiErrors',
            'validation'
        );
        errorTests.push(apiErrorResult);

        // Test network errors
        const networkErrorResult = await enhancedCommentFormIntegrationFactory.testErrorScenarios(
            'networkErrors', 
            'timeout'
        );
        errorTests.push(networkErrorResult);

        return errorTests;
    },

    /**
     * Test performance with baseline validation
     */
    async testPerformance() {
        const baseline = {
            maxExecutionTime: 2000, // 2 seconds
            maxMemoryUsage: 50 * 1024 * 1024, // 50MB
            maxDbQueries: 10,
            minSuccessRate: 0.95,
        };

        return await enhancedTestUtils.benchmark(
            () => enhancedCommentFormIntegrationFactory.testRoundTripSubmission('minimal'),
            baseline,
            { iterations: 10, concurrency: 3 }
        );
    },
};