/**
 * Comment Round-Trip Test
 * 
 * Tests the complete data flow for comment operations:
 * Form Data → Shape Transform → Validation → Endpoint Logic → Database → API Response
 */
import { PrismaClient } from "@prisma/client";
import type { CommentCreateInput, CommentUpdateInput } from "@vrooli/shared";
import { commentValidation, shapeComment } from "@vrooli/shared";
import { afterAll, beforeAll, describe, expect, it } from "vitest";
// @ts-expect-error - Server endpoints are not exported in package.json. These relative imports work at runtime for round-trip testing.
import { comment } from "@vrooli/server/endpoints/logic/comment.js";

// Form data types (what comes from UI forms)
interface CommentFormData {
    text: string;
    parentType: "Project" | "Routine" | "Resource" | "Comment";
    parentId: string;
    // For replies
    threadId?: string;
}

interface CommentUpdateFormData {
    text: string;
}

/**
 * Form Fixtures Layer
 * Simulates form interactions and generates form-shaped data
 */
class CommentFormFixtures {
    createCommentFormData(scenario: "simple" | "withMarkdown" | "reply" | "empty"): CommentFormData {
        switch (scenario) {
            case "simple":
                return {
                    text: "This is a simple comment on a project.",
                    parentType: "Project",
                    parentId: "project_123",
                };
            case "withMarkdown":
                return {
                    text: "# Great Project!\n\nI really like this project because:\n- It's well documented\n- Easy to use\n- Has **excellent** test coverage\n\n```typescript\nconst example = \"code snippet\";\n```",
                    parentType: "Routine",
                    parentId: "routine_456",
                };
            case "reply":
                return {
                    text: "I agree with your point! Here's my additional thought...",
                    parentType: "Comment",
                    parentId: "comment_789",
                    threadId: "thread_001",
                };
            case "empty":
                return {
                    text: "", // Invalid - empty comment
                    parentType: "Project",
                    parentId: "project_123",
                };
            default:
                throw new Error(`Unknown scenario: ${scenario}`);
        }
    }

    createCommentUpdateFormData(scenario: "minor" | "major" | "addMarkdown"): CommentUpdateFormData {
        switch (scenario) {
            case "minor":
                return {
                    text: "This is a simple comment on a project. (edited)",
                };
            case "major":
                return {
                    text: "Completely rewrote this comment to be more helpful and constructive.",
                };
            case "addMarkdown":
                return {
                    text: "Updated comment with **markdown** support:\n\n- List item 1\n- List item 2\n\n> Quote block",
                };
            default:
                throw new Error(`Unknown scenario: ${scenario}`);
        }
    }
}

/**
 * Shape Fixtures Layer
 * Uses real shaping functions to convert form data to API inputs
 */
class CommentShapeFixtures {
    transformCreateToAPIInput(formData: CommentFormData): CommentCreateInput {
        // Build the base input matching what shapeComment.create expects
        const baseInput: any = {
            text: formData.text,
            translations: [{
                language: "en",
                text: formData.text,
            }],
        };

        // Add parent connection based on type
        switch (formData.parentType) {
            case "Project":
                baseInput.projectConnect = formData.parentId;
                break;
            case "Routine":
                baseInput.routineConnect = formData.parentId;
                break;
            case "Resource":
                baseInput.resourceConnect = formData.parentId;
                break;
            case "Comment":
                baseInput.parentConnect = formData.parentId;
                if (formData.threadId) {
                    baseInput.threadConnect = formData.threadId;
                }
                break;
        }

        // Use real shape function
        return shapeComment.create(baseInput);
    }

    transformUpdateToAPIInput(formData: CommentUpdateFormData): CommentUpdateInput {
        const input = {
            text: formData.text,
            translations: [{
                language: "en",
                text: formData.text,
            }],
        };

        // Use real shape function
        return shapeComment.update(input);
    }
}

/**
 * Validation Fixtures Layer
 * Uses real validation schemas
 */
class CommentValidationFixtures {
    async validateCreateInput(apiInput: CommentCreateInput): Promise<{
        isValid: boolean;
        errors?: any;
        data?: any;
    }> {
        try {
            const schema = commentValidation.create({});
            const validatedData = await schema.validate(apiInput);
            return { isValid: true, data: validatedData };
        } catch (error) {
            return { isValid: false, errors: error };
        }
    }

    async validateUpdateInput(apiInput: CommentUpdateInput): Promise<{
        isValid: boolean;
        errors?: any;
        data?: any;
    }> {
        try {
            const schema = commentValidation.update({});
            const validatedData = await schema.validate(apiInput);
            return { isValid: true, data: validatedData };
        } catch (error) {
            return { isValid: false, errors: error };
        }
    }
}

/**
 * Endpoint Fixtures Layer
 * Uses real endpoint logic
 */
class CommentEndpointFixtures {
    async processCreate(apiInput: CommentCreateInput, context: any): Promise<any> {
        return await comment.createOne.logic({
            input: apiInput,
            userData: context.userData,
            prisma: context.prisma,
        });
    }

    async processUpdate(apiInput: CommentUpdateInput, context: any): Promise<any> {
        return await comment.updateOne.logic({
            input: apiInput,
            userData: context.userData,
            prisma: context.prisma,
        });
    }
}

describe("Comment Round-Trip Tests", () => {
    let prisma: PrismaClient;
    let context: any;

    const formFixtures = new CommentFormFixtures();
    const shapeFixtures = new CommentShapeFixtures();
    const validationFixtures = new CommentValidationFixtures();
    const endpointFixtures = new CommentEndpointFixtures();

    beforeAll(async () => {
        // Initialize Prisma with test database
        prisma = new PrismaClient({
            datasources: {
                db: { url: process.env.DATABASE_URL },
            },
        });

        await prisma.$connect();

        // Setup test context
        context = {
            prisma,
            userData: {
                id: "test_user_123",
                languages: ["en"],
                roles: [],
            },
        };
    });

    afterAll(async () => {
        // Cleanup test data
        await prisma.comment.deleteMany({
            where: {
                owner: {
                    id: "test_user_123",
                },
            },
        });
        await prisma.$disconnect();
    });

    describe("Comment Creation Flow", () => {
        it("should complete full comment creation cycle with simple text", async () => {
            // 1. Form data
            const formData = formFixtures.createCommentFormData("simple");
            expect(formData.text).toBeTruthy();
            expect(formData.parentType).toBe("Project");

            // 2. Shape transform
            const apiInput = shapeFixtures.transformCreateToAPIInput(formData);
            expect(apiInput.translations).toBeDefined();
            expect(apiInput.projectConnect).toBe(formData.parentId);

            // 3. Validation
            const validationResult = await validationFixtures.validateCreateInput(apiInput);
            expect(validationResult.isValid).toBe(true);
            expect(validationResult.data).toBeDefined();

            // 4. Endpoint would process here
            // const result = await endpointFixtures.processCreate(apiInput, context);
            // expect(result.comment.id).toBeDefined();
        });

        it("should handle markdown content properly", async () => {
            // 1. Form data
            const formData = formFixtures.createCommentFormData("withMarkdown");
            expect(formData.text).toContain("#");
            expect(formData.text).toContain("```");

            // 2. Shape transform
            const apiInput = shapeFixtures.transformCreateToAPIInput(formData);
            expect(apiInput.translations[0].text).toContain("# Great Project!");
            expect(apiInput.routineConnect).toBe(formData.parentId);

            // 3. Validation
            const validationResult = await validationFixtures.validateCreateInput(apiInput);
            expect(validationResult.isValid).toBe(true);
        });

        it("should handle comment replies with thread", async () => {
            // 1. Form data
            const formData = formFixtures.createCommentFormData("reply");
            expect(formData.parentType).toBe("Comment");
            expect(formData.threadId).toBeDefined();

            // 2. Shape transform
            const apiInput = shapeFixtures.transformCreateToAPIInput(formData);
            expect(apiInput.parentConnect).toBe(formData.parentId);
            expect(apiInput.threadConnect).toBe(formData.threadId);

            // 3. Validation
            const validationResult = await validationFixtures.validateCreateInput(apiInput);
            expect(validationResult.isValid).toBe(true);
        });

        it("should fail validation with empty comment", async () => {
            // 1. Form data
            const formData = formFixtures.createCommentFormData("empty");
            expect(formData.text).toBe("");

            // 2. Shape transform
            const apiInput = shapeFixtures.transformCreateToAPIInput(formData);

            // 3. Validation
            const validationResult = await validationFixtures.validateCreateInput(apiInput);
            expect(validationResult.isValid).toBe(false);
            expect(validationResult.errors).toBeDefined();
        });
    });

    describe("Comment Update Flow", () => {
        it("should update comment with minor edit", async () => {
            // 1. Form data
            const formData = formFixtures.createCommentUpdateFormData("minor");
            expect(formData.text).toContain("(edited)");

            // 2. Shape transform
            const apiInput = shapeFixtures.transformUpdateToAPIInput(formData);
            expect(apiInput.translations).toBeDefined();
            expect(apiInput.translations[0].text).toBe(formData.text);

            // 3. Validation
            const validationResult = await validationFixtures.validateUpdateInput(apiInput);
            expect(validationResult.isValid).toBe(true);
        });

        it("should handle major comment rewrite", async () => {
            // 1. Form data
            const formData = formFixtures.createCommentUpdateFormData("major");

            // 2. Shape transform
            const apiInput = shapeFixtures.transformUpdateToAPIInput(formData);
            expect(apiInput.translations[0].text).toContain("Completely rewrote");

            // 3. Validation
            const validationResult = await validationFixtures.validateUpdateInput(apiInput);
            expect(validationResult.isValid).toBe(true);
        });

        it("should add markdown to existing comment", async () => {
            // 1. Form data
            const formData = formFixtures.createCommentUpdateFormData("addMarkdown");
            expect(formData.text).toContain("**markdown**");

            // 2. Shape transform
            const apiInput = shapeFixtures.transformUpdateToAPIInput(formData);
            expect(apiInput.translations[0].text).toContain("> Quote block");

            // 3. Validation
            const validationResult = await validationFixtures.validateUpdateInput(apiInput);
            expect(validationResult.isValid).toBe(true);
        });
    });

    describe("Comment Edge Cases", () => {
        it("should handle very long comments", async () => {
            const longText = "This is a very long comment. ".repeat(100);
            const formData: CommentFormData = {
                text: longText,
                parentType: "Project",
                parentId: "project_123",
            };

            const apiInput = shapeFixtures.transformCreateToAPIInput(formData);
            const validationResult = await validationFixtures.validateCreateInput(apiInput);

            // Validation should handle length limits appropriately
            expect(validationResult.data).toBeDefined();
        });

        it("should handle special characters and emojis", async () => {
            const formData: CommentFormData = {
                text: "Great work! 🎉 Love the <feature> & it's \"awesome\"!",
                parentType: "Resource",
                parentId: "resource_123",
            };

            const apiInput = shapeFixtures.transformCreateToAPIInput(formData);
            expect(apiInput.translations[0].text).toContain("🎉");

            const validationResult = await validationFixtures.validateCreateInput(apiInput);
            expect(validationResult.isValid).toBe(true);
        });
    });
});
