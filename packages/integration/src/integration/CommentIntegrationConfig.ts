/**
 * Comment Integration Configuration - Using Form Fixtures Only
 * 
 * This demonstrates the new approach: use UIFormTestConfig transformation methods
 * directly instead of separate ApiInputTransformer. This eliminates duplication
 * and keeps all form-related logic in one place.
 * 
 * The key insight: UIFormTestConfig already has all needed transformation methods!
 */

import type {
    Comment,
    CommentShape,
    CommentCreateInput,
    CommentUpdateInput,
    Session,
} from "@vrooli/shared";
import type {
    StandardIntegrationConfig,
    EndpointCaller,
    DatabaseVerifier,
} from "./types.js";
// Import the existing UI form test configuration with integration support
// This contains all the form fixtures, transformations, and response handlers
import { commentFormTestConfig } from "@vrooli/ui/src/__test/fixtures/form-testing/examples/CommentFormTestWithIntegration.js";
import { getPrisma } from "../setup/test-setup.js";

/**
 * Note: API Input Transformer logic has been moved to commentFormTestConfig.
 * The UIFormTestConfig now includes:
 * - responseToCreateInput
 * - responseToUpdateInput
 * - validateBidirectionalTransform
 * - responseToFormData
 * 
 * This eliminates the need for a separate transformer and keeps all
 * transformation logic together in the form configuration.
 */

/**
 * Endpoint Caller for Comments
 * 
 * Calls actual comment endpoints - this is where the real API testing happens.
 */
const commentEndpointCaller: EndpointCaller<CommentCreateInput, CommentUpdateInput, Comment> = {
    create: async (input: CommentCreateInput, session?: Session) => {
        const startTime = Date.now();

        try {
            // Import and call actual endpoint logic
            const { default: commentLogic } = await import("@vrooli/server/src/endpoints/logic/comment.js");

            const result = await commentLogic.Create.performLogic({
                input,
                userData: session || null,
            });

            return {
                success: true,
                data: result,
                timing: Date.now() - startTime,
            };
        } catch (error) {
            return {
                success: false,
                error: {
                    code: "CREATE_FAILED",
                    message: error instanceof Error ? error.message : "Unknown error",
                    statusCode: 500,
                },
                timing: Date.now() - startTime,
            };
        }
    },

    update: async (id: string, input: CommentUpdateInput, session?: Session) => {
        const startTime = Date.now();

        try {
            const { default: commentLogic } = await import("@vrooli/server/src/endpoints/logic/comment.js");

            const result = await commentLogic.Update.performLogic({
                input: { ...input, id },
                userData: session || null,
            });

            return {
                success: true,
                data: result,
                timing: Date.now() - startTime,
            };
        } catch (error) {
            return {
                success: false,
                error: {
                    code: "UPDATE_FAILED",
                    message: error instanceof Error ? error.message : "Unknown error",
                    statusCode: 500,
                },
                timing: Date.now() - startTime,
            };
        }
    },

    read: async (id: string, session?: Session) => {
        const startTime = Date.now();

        try {
            const { default: commentLogic } = await import("@vrooli/server/src/endpoints/logic/comment.js");

            const result = await commentLogic.Read.performLogic({
                input: { id },
                userData: session || null,
            });

            return {
                success: true,
                data: result,
                timing: Date.now() - startTime,
            };
        } catch (error) {
            return {
                success: false,
                error: {
                    code: "READ_FAILED",
                    message: error instanceof Error ? error.message : "Unknown error",
                    statusCode: 404,
                },
                timing: Date.now() - startTime,
            };
        }
    },

    delete: async (id: string, session?: Session) => {
        const startTime = Date.now();

        try {
            const { default: commentLogic } = await import("@vrooli/server/src/endpoints/logic/comment.js");

            await commentLogic.Delete.performLogic({
                input: { id },
                userData: session || null,
            });

            return {
                success: true,
                timing: Date.now() - startTime,
            };
        } catch (error) {
            return {
                success: false,
                error: {
                    code: "DELETE_FAILED",
                    message: error instanceof Error ? error.message : "Unknown error",
                    statusCode: 500,
                },
                timing: Date.now() - startTime,
            };
        }
    },
};

/**
 * Database Verifier for Comments
 * 
 * Direct database access to verify persistence.
 */
const commentDatabaseVerifier: DatabaseVerifier<Comment> = {
    findById: async (id: string): Promise<Comment | null> => {
        try {
            const prisma = getPrisma();
            const comment = await prisma.comment.findUnique({
                where: { id },
                include: {
                    translations: true,
                    creator: true,
                    // Add other includes as needed
                },
            });

            if (!comment) return null;

            // Convert Prisma result to Comment type
            return {
                ...comment,
                __typename: "Comment" as const,
                // Map other fields as needed based on your Comment type
            } as Comment;

        } catch (error) {
            console.error("Database find error:", error);
            return null;
        }
    },

    findByConstraints: async (_constraints: Record<string, any>) => {
        // Implementation for finding by constraints
        return null; // Simplified for example
    },

    verifyConsistency: (apiResult: Comment, databaseResult: Comment) => {
        const differences: Array<{ field: string; apiValue: any; dbValue: any }> = [];

        // Check essential fields
        if (apiResult.id !== databaseResult.id) {
            differences.push({ field: "id", apiValue: apiResult.id, dbValue: databaseResult.id });
        }

        return {
            consistent: differences.length === 0,
            differences,
        };
    },

    cleanup: async (id: string) => {
        try {
            const prisma = getPrisma();
            await prisma.comment.delete({ where: { id } });
        } catch (error) {
            console.error("Cleanup error:", error);
        }
    },
};

/**
 * Complete Comment Integration Configuration
 * 
 * This is now even simpler - we just combine the existing UI form config 
 * (which includes all transformation methods) with endpoint caller and database verifier.
 */
export const commentIntegrationConfig: StandardIntegrationConfig<
    any, // Form data type from UI fixtures
    CommentShape,
    CommentCreateInput,
    CommentUpdateInput,
    Comment
> = {
    objectType: "Comment",
    uiFormConfig: commentFormTestConfig,  // ← Contains all transformation methods now!
    endpointCaller: commentEndpointCaller,
    databaseVerifier: commentDatabaseVerifier,
    validation: commentFormTestConfig.validation, // Already included in UI config
};

// Export for individual use
export {
    commentEndpointCaller,
    commentDatabaseVerifier,
};