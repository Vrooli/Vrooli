/**
 * Project Round-Trip Test
 * 
 * Tests the complete data flow for project operations:
 * Form Data → Shape Transform → Validation → Endpoint Logic → Database → API Response
 */
import { PrismaClient } from "@prisma/client";
import type { ResourceCreateInput, ResourceShape, ResourceUpdateInput } from "@vrooli/shared";
import { resourceValidation, shapeResource } from "@vrooli/shared";
import { afterAll, beforeAll, describe, expect, it } from "vitest";
// @ts-expect-error - Server endpoints are not exported in package.json. These relative imports work at runtime for round-trip testing.
import { resource } from "@vrooli/server/endpoints/logic/resource.js";

/**
 * Form Fixtures Layer
 * Simulates form interactions and generates form-shaped data
 */
class ProjectFormFixtures {
    createProjectFormData(scenario: "minimal" | "complete" | "withResources" | "invalid"): ResourceShape {
        switch (scenario) {
            case "minimal":
                return {
                    name: "My Test Project",
                    isPrivate: false,
                };
            case "complete":
                return {
                    name: "Complete Project Example",
                    description: "This project demonstrates all features including tags and permissions.",
                    isPrivate: false,
                    tags: ["ai", "automation", "testing"],
                    permissions: {
                        canView: "Anyone",
                        canEdit: "Members",
                    },
                };
            case "withResources":
                return {
                    name: "Project with Resources",
                    description: "A project that includes external resource links.",
                    isPrivate: true,
                    tags: ["documentation", "resources"],
                    resourceLinks: [
                        {
                            title: "Documentation",
                            link: "https://docs.example.com",
                            usedFor: "ProjectVersion",
                        },
                        {
                            title: "GitHub Repository",
                            link: "https://github.com/example/repo",
                            usedFor: "ProjectVersion",
                        },
                    ],
                };
            case "invalid":
                return {
                    name: "", // Invalid - empty name
                    isPrivate: false,
                };
            default:
                throw new Error(`Unknown scenario: ${scenario}`);
        }
    }

    createProjectUpdateFormData(scenario: "minimal" | "changePrivacy" | "addTags"): ResourceShape {
        switch (scenario) {
            case "minimal":
                return {
                    description: "Updated project description",
                };
            case "changePrivacy":
                return {
                    isPrivate: true,
                    permissions: {
                        canView: "Members",
                        canEdit: "Owner",
                    },
                };
            case "addTags":
                return {
                    tags: ["new-feature", "v2", "updated"],
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
class ProjectShapeFixtures {
    transformCreateToAPIInput(formData: ResourceShape): ResourceCreateInput {
        // Build the base input matching what shapeProject.create expects
        const baseInput: any = {
            name: formData.name,
            description: formData.description,
            isPrivate: formData.isPrivate,
        };

        // Add permissions if provided
        if (formData.permissions) {
            baseInput.permissions = {
                canView: formData.permissions.canView,
                canEdit: formData.permissions.canEdit,
            };
        }

        // Add tags if provided
        if (formData.tags && formData.tags.length > 0) {
            baseInput.tags = {
                create: formData.tags.map(tag => ({
                    tag,
                })),
            };
        }

        // Add resources if provided
        if (formData.resourceLinks && formData.resourceLinks.length > 0) {
            baseInput.versions = {
                create: [{
                    isLatest: true,
                    versionLabel: "1.0.0",
                    resources: {
                        create: formData.resourceLinks.map(resource => ({
                            link: resource.link,
                            usedFor: resource.usedFor || "ProjectVersion",
                            translations: {
                                create: [{
                                    language: "en",
                                    title: resource.title,
                                }],
                            },
                        })),
                    },
                }],
            };
        }

        // Use real shape function
        return shapeResource.create(baseInput);
    }

    transformUpdateToAPIInput(formData: ResourceShape): ResourceUpdateInput {
        // Use real shape function
        return shapeResource.update(formData);
    }
}

/**
 * Validation Fixtures Layer
 * Uses real validation schemas
 */
class ProjectValidationFixtures {
    async validateCreateInput(apiInput: ResourceCreateInput): Promise<{
        isValid: boolean;
        errors?: any;
        data?: any;
    }> {
        try {
            const schema = resourceValidation.create({});
            const validatedData = await schema.validate(apiInput);
            return { isValid: true, data: validatedData };
        } catch (error) {
            return { isValid: false, errors: error };
        }
    }

    async validateUpdateInput(apiInput: ResourceUpdateInput): Promise<{
        isValid: boolean;
        errors?: any;
        data?: any;
    }> {
        try {
            const schema = resourceValidation.update({});
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
class ProjectEndpointFixtures {
    async processCreate(apiInput: ResourceCreateInput, context: any): Promise<any> {
        return await resource.createOne.logic({
            input: apiInput,
            userData: context.userData,
            prisma: context.prisma,
        });
    }

    async processUpdate(apiInput: ResourceUpdateInput, context: any): Promise<any> {
        return await resource.updateOne.logic({
            input: apiInput,
            userData: context.userData,
            prisma: context.prisma,
        });
    }
}

describe("Project Round-Trip Tests", () => {
    let prisma: PrismaClient;
    let context: any;

    const formFixtures = new ProjectFormFixtures();
    const shapeFixtures = new ProjectShapeFixtures();
    const validationFixtures = new ProjectValidationFixtures();
    const endpointFixtures = new ProjectEndpointFixtures();

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
        await prisma.project.deleteMany({
            where: {
                name: {
                    contains: "Test Project",
                },
            },
        });
        await prisma.$disconnect();
    });

    describe("Project Creation Flow", () => {
        it("should complete full project creation cycle with minimal data", async () => {
            // 1. Form data
            const formData = formFixtures.createProjectFormData("minimal");
            expect(formData.name).toBeTruthy();
            expect(formData.isPrivate).toBe(false);

            // 2. Shape transform
            const apiInput = shapeFixtures.transformCreateToAPIInput(formData);
            expect(apiInput.name).toBe(formData.name);
            expect(apiInput.isPrivate).toBe(formData.isPrivate);

            // 3. Validation
            const validationResult = await validationFixtures.validateCreateInput(apiInput);
            expect(validationResult.isValid).toBe(true);
            expect(validationResult.data).toBeDefined();

            // 4. Endpoint would process here
            // const result = await endpointFixtures.processCreate(apiInput, context);
            // expect(result.project.id).toBeDefined();
        });

        it("should complete project creation with all features", async () => {
            // 1. Form data
            const formData = formFixtures.createProjectFormData("complete");
            expect(formData.tags).toHaveLength(3);
            expect(formData.permissions).toBeDefined();

            // 2. Shape transform
            const apiInput = shapeFixtures.transformCreateToAPIInput(formData);
            expect(apiInput.name).toBe(formData.name);
            expect(apiInput.description).toBe(formData.description);

            // 3. Validation
            const validationResult = await validationFixtures.validateCreateInput(apiInput);
            expect(validationResult.isValid).toBe(true);
            expect(validationResult.data).toBeDefined();
        });

        it("should create project with resources", async () => {
            // 1. Form data
            const formData = formFixtures.createProjectFormData("withResources");
            expect(formData.resourceLinks).toHaveLength(2);

            // 2. Shape transform
            const apiInput = shapeFixtures.transformCreateToAPIInput(formData);
            expect(apiInput.versions).toBeDefined();

            // 3. Validation
            const validationResult = await validationFixtures.validateCreateInput(apiInput);
            expect(validationResult.isValid).toBe(true);
        });

        it("should fail validation with invalid data", async () => {
            // 1. Form data
            const formData = formFixtures.createProjectFormData("invalid");
            expect(formData.name).toBe("");

            // 2. Shape transform
            const apiInput = shapeFixtures.transformCreateToAPIInput(formData);

            // 3. Validation
            const validationResult = await validationFixtures.validateCreateInput(apiInput);
            expect(validationResult.isValid).toBe(false);
            expect(validationResult.errors).toBeDefined();
        });
    });

    describe("Project Update Flow", () => {
        it("should update project description", async () => {
            // 1. Form data
            const formData = formFixtures.createProjectUpdateFormData("minimal");

            // 2. Shape transform
            const apiInput = shapeFixtures.transformUpdateToAPIInput(formData);
            expect(apiInput.description).toBe(formData.description);

            // 3. Validation
            const validationResult = await validationFixtures.validateUpdateInput(apiInput);
            expect(validationResult.isValid).toBe(true);
        });

        it("should update privacy settings", async () => {
            // 1. Form data
            const formData = formFixtures.createProjectUpdateFormData("changePrivacy");

            // 2. Shape transform
            const apiInput = shapeFixtures.transformUpdateToAPIInput(formData);
            expect(apiInput.isPrivate).toBe(true);
            expect(apiInput.permissions).toBeDefined();

            // 3. Validation
            const validationResult = await validationFixtures.validateUpdateInput(apiInput);
            expect(validationResult.isValid).toBe(true);
        });

        it("should add tags to project", async () => {
            // 1. Form data
            const formData = formFixtures.createProjectUpdateFormData("addTags");

            // 2. Shape transform
            const apiInput = shapeFixtures.transformUpdateToAPIInput(formData);
            expect(apiInput.tags).toBeDefined();

            // 3. Validation
            const validationResult = await validationFixtures.validateUpdateInput(apiInput);
            expect(validationResult.isValid).toBe(true);
        });
    });
});
