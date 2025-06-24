/**
 * Bookmark Form Integration Testing Example
 * 
 * This demonstrates the migration from legacy round-trip tests to the modern
 * form-testing infrastructure for Bookmark operations. This provides true
 * round-trip testing through real API endpoints and database persistence.
 */

import { 
    type Bookmark, 
    type BookmarkCreateInput, 
    type BookmarkUpdateInput, 
    type BookmarkShape,
    bookmarkValidation,
    shapeBookmark,
    endpointsBookmark,
    BookmarkFor,
    DUMMY_ID,
} from "@vrooli/shared";
import { createIntegrationFormTestFactory } from "../engine/IntegrationFormTestFactory.js";
import { getPrisma } from "../../setup/test-setup.js";

/**
 * Form data type for bookmark testing (simulates UI form data)
 */
interface BookmarkFormData {
    bookmarkFor: BookmarkFor;
    forConnect?: string;  // ID of existing object to bookmark
    list?: {
        id: string;      // "new" for new list, or existing list ID
        label?: string;  // For new lists
    };
    createNewList?: boolean;
    newListLabel?: string;
}

/**
 * Test fixtures for different bookmark scenarios
 */
export const bookmarkFormFixtures: Record<string, BookmarkFormData> = {
    minimal: {
        bookmarkFor: BookmarkFor.User,
        forConnect: "test-user-id",
        list: {
            id: "existing-list-id",
        },
    },
    
    withNewList: {
        bookmarkFor: BookmarkFor.Project,
        forConnect: "test-project-id",
        createNewList: true,
        newListLabel: "My Favorite Projects",
        list: {
            id: "new",
            label: "My Favorite Projects",
        },
    },
    
    complete: {
        bookmarkFor: BookmarkFor.Routine,
        forConnect: "test-routine-id",
        list: {
            id: "comprehensive-list-id",
        },
    },
    
    edgeCase: {
        bookmarkFor: BookmarkFor.Standard,
        forConnect: "test-standard-id",
        list: {
            id: "new",
            label: "A".repeat(255), // Maximum length label
        },
    },
    
    invalid: {
        bookmarkFor: BookmarkFor.User,
        // Missing forConnect - should fail validation
        list: {
            id: "test-list-id",
        },
    },
};

/**
 * Convert form data to BookmarkShape
 */
function bookmarkFormToShape(formData: BookmarkFormData): BookmarkShape {
    const shape: BookmarkShape = {
        __typename: "Bookmark",
        id: DUMMY_ID,
        bookmarkFor: formData.bookmarkFor,
    };

    // Handle object being bookmarked
    if (formData.forConnect) {
        switch (formData.bookmarkFor) {
            case BookmarkFor.User:
                shape.to = {
                    __typename: "User",
                    id: formData.forConnect,
                };
                break;
            case BookmarkFor.Project:
                shape.to = {
                    __typename: "Project", 
                    id: formData.forConnect,
                };
                break;
            case BookmarkFor.Routine:
                shape.to = {
                    __typename: "Routine",
                    id: formData.forConnect,
                };
                break;
            case BookmarkFor.Standard:
                shape.to = {
                    __typename: "Standard",
                    id: formData.forConnect,
                };
                break;
        }
    }

    // Handle bookmark list
    if (formData.list) {
        if (formData.list.id === "new" && formData.list.label) {
            shape.list = {
                __typename: "BookmarkList",
                id: DUMMY_ID,
                label: formData.list.label,
            };
        } else {
            shape.list = {
                __typename: "BookmarkList",
                id: formData.list.id,
            };
        }
    }

    return shape;
}

/**
 * Transform bookmark values for API calls
 */
function transformBookmarkValues(values: BookmarkShape, existing: BookmarkShape, isCreate: boolean): BookmarkCreateInput | BookmarkUpdateInput {
    return isCreate ? shapeBookmark.create(values) : shapeBookmark.update(existing, values);
}

/**
 * Find bookmark in database
 */
async function findBookmarkInDatabase(id: string): Promise<Bookmark | null> {
    const prisma = getPrisma();
    if (!prisma) return null;
    
    try {
        return await prisma.bookmark.findUnique({
            where: { id },
            include: {
                list: true,
                to: true,
                by: true,
            },
        });
    } catch (error) {
        console.error("Error finding bookmark in database:", error);
        return null;
    }
}

/**
 * Integration test factory for Bookmark forms
 */
export const bookmarkFormIntegrationFactory = createIntegrationFormTestFactory({
    objectType: "Bookmark",
    validation: bookmarkValidation,
    transformFunction: transformBookmarkValues,
    endpoints: {
        create: endpointsBookmark.createOne,
        update: endpointsBookmark.updateOne,
    },
    formFixtures: bookmarkFormFixtures,
    formToShape: bookmarkFormToShape,
    findInDatabase: findBookmarkInDatabase,
    prismaModel: "bookmark",
});

/**
 * Test cases for bookmark form integration
 */
export const bookmarkIntegrationTestCases = bookmarkFormIntegrationFactory.generateIntegrationTestCases();

/**
 * Helper function to create test bookmark targets and lists
 */
export async function createTestBookmarkTarget(type: BookmarkFor): Promise<string> {
    const prisma = getPrisma();
    if (!prisma) throw new Error("Prisma not available");

    // Create a minimal user first (needed for most objects)
    const user = await prisma.user.create({
        data: {
            id: `test-user-${Date.now()}`,
            name: "Test User",
            emails: {
                create: {
                    emailAddress: `test-${Date.now()}@example.com`,
                    verified: true,
                },
            },
        },
    });

    switch (type) {
        case BookmarkFor.User:
            return user.id;

        case BookmarkFor.Project:
            const project = await prisma.project.create({
                data: {
                    id: `test-project-${Date.now()}`,
                    isPrivate: false,
                    creator: { connect: { id: user.id } },
                    owner: { connect: { id: user.id } },
                    translations: {
                        create: {
                            language: "en",
                            name: "Test Project",
                            description: "A test project for bookmark integration testing",
                        },
                    },
                },
            });
            return project.id;

        case BookmarkFor.Routine:
            const routine = await prisma.routine.create({
                data: {
                    id: `test-routine-${Date.now()}`,
                    isInternal: false,
                    isPrivate: false,
                    creator: { connect: { id: user.id } },
                    owner: { connect: { id: user.id } },
                    versions: {
                        create: {
                            id: `test-routine-version-${Date.now()}`,
                            versionLabel: "1.0.0",
                            isComplete: true,
                            isPrivate: false,
                            creator: { connect: { id: user.id } },
                            root: { connect: { id: user.id } },
                            translations: {
                                create: {
                                    language: "en",
                                    name: "Test Routine",
                                    description: "A test routine for bookmark integration testing",
                                },
                            },
                        },
                    },
                },
            });
            return routine.id;

        case BookmarkFor.Standard:
            const standard = await prisma.standard.create({
                data: {
                    id: `test-standard-${Date.now()}`,
                    isPrivate: false,
                    creator: { connect: { id: user.id } },
                    owner: { connect: { id: user.id } },
                    versions: {
                        create: {
                            id: `test-standard-version-${Date.now()}`,
                            versionLabel: "1.0.0",
                            isComplete: true,
                            isPrivate: false,
                            creator: { connect: { id: user.id } },
                            root: { connect: { id: user.id } },
                            translations: {
                                create: {
                                    language: "en",
                                    name: "Test Standard",
                                    description: "A test standard for bookmark integration testing",
                                },
                            },
                        },
                    },
                },
            });
            return standard.id;

        default:
            throw new Error(`Unsupported bookmark target type: ${type}`);
    }
}

/**
 * Helper function to create test bookmark lists
 */
export async function createTestBookmarkList(label: string, userId: string): Promise<string> {
    const prisma = getPrisma();
    if (!prisma) throw new Error("Prisma not available");

    const list = await prisma.bookmarkList.create({
        data: {
            id: `test-list-${Date.now()}`,
            label,
            user: { connect: { id: userId } },
        },
    });

    return list.id;
}