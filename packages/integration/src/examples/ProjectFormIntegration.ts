/**
 * Project Form Integration Testing Example
 * 
 * This demonstrates comprehensive round-trip testing for Project operations
 * using the IntegrationFormTestFactory. Projects are a core content type in
 * Vrooli for organizing and sharing collaborative work.
 */

import { 
    type Project, 
    type ProjectCreateInput, 
    type ProjectUpdateInput, 
    type ProjectShape,
    projectValidation,
    shapeProject,
    endpointsProject,
    DUMMY_ID,
} from "@vrooli/shared";
import { createIntegrationFormTestFactory } from "../engine/IntegrationFormTestFactory.js";
import { getPrisma } from "../../setup/test-setup.js";

/**
 * Form data type for project testing (simulates UI form data)
 */
interface ProjectFormData {
    name: string;
    description?: string;
    isPrivate: boolean;
    tags?: string[];
    // Project-specific fields
    handle?: string;
    translations?: Array<{
        language: string;
        name: string;
        description?: string;
    }>;
    // Team/ownership
    teamConnect?: string;
    // Resource management
    resourceLists?: Array<{
        label: string;
        description?: string;
    }>;
}

/**
 * Test fixtures for different project scenarios
 */
export const projectFormFixtures: Record<string, ProjectFormData> = {
    minimal: {
        name: "My Test Project",
        isPrivate: false,
    },
    
    complete: {
        name: "Complete Project Example",
        description: "This project demonstrates all features including tags, translations, and team collaboration.",
        isPrivate: false,
        handle: "complete-project-example",
        tags: ["ai", "automation", "testing", "collaboration"],
        translations: [
            { 
                language: "en", 
                name: "Complete Project Example", 
                description: "This project demonstrates all features including tags, translations, and team collaboration."
            },
            { 
                language: "es", 
                name: "Ejemplo de Proyecto Completo", 
                description: "Este proyecto demuestra todas las características incluyendo etiquetas, traducciones y colaboración en equipo."
            },
        ],
        resourceLists: [
            {
                label: "Documentation",
                description: "Project documentation and guides",
            },
            {
                label: "Code Examples",
                description: "Code samples and implementations",
            },
        ],
    },
    
    teamProject: {
        name: "Team Collaboration Project",
        description: "A project designed for team collaboration and shared ownership.",
        isPrivate: true,
        handle: "team-collab-project",
        tags: ["team", "collaboration", "private"],
        teamConnect: "test-team-id",
    },
    
    edgeCase: {
        name: "A".repeat(255), // Maximum length name
        description: "B".repeat(2048), // Long description
        isPrivate: false,
        tags: Array.from({ length: 20 }, (_, i) => `tag${i}`), // Many tags
    },
    
    invalid: {
        name: "", // Empty name should fail validation
        isPrivate: false,
    },
    
    longName: {
        name: "A".repeat(300), // Exceeds maximum length
        isPrivate: false,
    },
};

/**
 * Convert form data to ProjectShape
 */
function projectFormToShape(formData: ProjectFormData): ProjectShape {
    const shape: ProjectShape = {
        __typename: "Project",
        id: DUMMY_ID,
        isPrivate: formData.isPrivate,
        translations: formData.translations || [{
            __typename: "ProjectTranslation",
            id: DUMMY_ID,
            language: "en",
            name: formData.name,
            description: formData.description || "",
        }],
    };

    // Add handle if provided
    if (formData.handle) {
        shape.handle = formData.handle;
    }

    // Add tags if provided
    if (formData.tags && formData.tags.length > 0) {
        shape.tags = formData.tags.map(tag => ({
            __typename: "Tag",
            id: DUMMY_ID,
            tag,
        }));
    }

    // Add team connection if provided
    if (formData.teamConnect) {
        shape.team = {
            __typename: "Team",
            id: formData.teamConnect,
        };
    }

    // Add resource lists if provided
    if (formData.resourceLists && formData.resourceLists.length > 0) {
        shape.resourceLists = formData.resourceLists.map(list => ({
            __typename: "ResourceList",
            id: DUMMY_ID,
            translations: [{
                __typename: "ResourceListTranslation",
                id: DUMMY_ID,
                language: "en",
                name: list.label,
                description: list.description || "",
            }],
        }));
    }

    return shape;
}

/**
 * Transform project values for API calls
 */
function transformProjectValues(values: ProjectShape, existing: ProjectShape, isCreate: boolean): ProjectCreateInput | ProjectUpdateInput {
    return isCreate ? shapeProject.create(values) : shapeProject.update(existing, values);
}

/**
 * Find project in database
 */
async function findProjectInDatabase(id: string): Promise<Project | null> {
    const prisma = getPrisma();
    if (!prisma) return null;
    
    try {
        return await prisma.project.findUnique({
            where: { id },
            include: {
                translations: true,
                tags: true,
                team: true,
                owner: true,
                creator: true,
                resourceLists: {
                    include: {
                        translations: true,
                    },
                },
                members: true,
            },
        });
    } catch (error) {
        console.error("Error finding project in database:", error);
        return null;
    }
}

/**
 * Integration test factory for Project forms
 */
export const projectFormIntegrationFactory = createIntegrationFormTestFactory({
    objectType: "Project",
    validation: projectValidation,
    transformFunction: transformProjectValues,
    endpoints: {
        create: endpointsProject.createOne,
        update: endpointsProject.updateOne,
    },
    formFixtures: projectFormFixtures,
    formToShape: projectFormToShape,
    findInDatabase: findProjectInDatabase,
    prismaModel: "project",
});

/**
 * Test cases for project form integration
 */
export const projectIntegrationTestCases = projectFormIntegrationFactory.generateIntegrationTestCases();

/**
 * Helper function to create test team for project ownership
 */
export async function createTestTeam(name: string, userId: string): Promise<string> {
    const prisma = getPrisma();
    if (!prisma) throw new Error("Prisma not available");

    const team = await prisma.team.create({
        data: {
            id: `test-team-${Date.now()}`,
            isPrivate: false,
            creator: { connect: { id: userId } },
            translations: {
                create: {
                    language: "en",
                    name: name,
                    bio: `A test team for project integration testing`,
                },
            },
            members: {
                create: {
                    user: { connect: { id: userId } },
                    role: "Owner",
                },
            },
        },
    });

    return team.id;
}

/**
 * Helper function to create test user for project operations
 */
export async function createTestProjectUser(): Promise<{ id: string; email: string }> {
    const prisma = getPrisma();
    if (!prisma) throw new Error("Prisma not available");

    const user = await prisma.user.create({
        data: {
            id: `test-user-${Date.now()}`,
            name: "Test Project User",
            emails: {
                create: {
                    emailAddress: `test-project-${Date.now()}@example.com`,
                    verified: true,
                },
            },
        },
        include: {
            emails: true,
        },
    });

    return {
        id: user.id,
        email: user.emails[0].emailAddress,
    };
}