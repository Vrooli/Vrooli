import { type ResourceCreateInput, type Resource, generatePK, ResourceType } from "@vrooli/shared";

/**
 * Mock service for simulating resource API operations in tests
 * This provides a simple in-memory storage for testing data transformations
 */

// Simple in-memory storage for test resources
const getStorage = () => {
    if (!(globalThis as any).__testResourceStorage) {
        (globalThis as any).__testResourceStorage = {};
    }
    return (globalThis as any).__testResourceStorage;
};

function createMockResource(createData: ResourceCreateInput): Resource {
    const now = new Date().toISOString();
    const version = createData.versionsCreate?.[0];
    const translation = version?.translationsCreate?.[0];
    
    return {
        __typename: "Resource",
        id: createData.id,
        resourceType: createData.resourceType,
        isPrivate: createData.isPrivate || false,
        isInternal: createData.isInternal || false,
        permissions: createData.permissions || JSON.stringify(["Read"]),
        handle: null,
        publicId: createData.publicId || null,
        createdAt: now,
        updatedAt: now,
        score: 0,
        views: 0,
        owner: {
            __typename: "User",
            id: "user_test_123456789",
            handle: "testuser",
            name: "Test User",
        },
        tags: createData.tagsConnect?.map(tag => {
            const tagName = typeof tag === 'string' ? tag : tag.tag;
            return {
                __typename: "Tag",
                id: `tag_${tagName}_123456789`,
                tag: tagName,
                created_at: now,
                bookmarks: 0,
                translations: [],
                you: {
                    __typename: "TagYou",
                    isBookmarked: false,
                },
            };
        }) || [],
        versions: version ? [{
            __typename: "ResourceVersion",
            id: version.id,
            versionLabel: version.versionLabel || "1.0.0",
            versionNotes: version.versionNotes || null,
            isLatest: true,
            isPrivate: version.isPrivate || false,
            isComplete: version.isComplete || false,
            resourceSubType: version.resourceSubType || null,
            createdAt: now,
            updatedAt: now,
            completedAt: null,
            translations: translation ? [{
                __typename: "ResourceVersionTranslation",
                id: translation.id,
                language: translation.language,
                name: translation.name,
                description: translation.description,
                details: translation.details || null,
                instructions: translation.instructions || null,
            }] : [],
            you: {
                __typename: "ResourceVersionYou",
                canComment: true,
                canDelete: false,
                canReport: true,
                canUpdate: false,
                canUse: true,
                canRead: true,
            },
        }] : [],
        versionsCount: 1,
        you: {
            __typename: "ResourceYou",
            canComment: true,
            canDelete: false,
            canBookmark: true,
            canReact: true,
            canRead: true,
            canReport: true,
            canUpdate: false,
            canTransfer: false,
            isBookmarked: false,
            reaction: null,
            isReported: false,
            isViewed: false,
        },
    };
}

export const mockResourceService = {
    async create(createData: ResourceCreateInput): Promise<Resource> {
        const storage = getStorage();
        const resource = createMockResource(createData);
        storage[resource.id] = resource;
        return resource;
    },

    async findById(id: string): Promise<Resource> {
        const storage = getStorage();
        const resource = storage[id];
        if (!resource) {
            throw new Error(`Resource with id ${id} not found`);
        }
        return resource;
    },

    async update(id: string, updateData: any): Promise<Resource> {
        const storage = getStorage();
        const existing = storage[id];
        if (!existing) {
            throw new Error(`Resource with id ${id} not found`);
        }

        const updated = {
            ...existing,
            ...updateData,
            updatedAt: new Date().toISOString(),
        };

        // Handle tag updates
        if (updateData.tagsConnect) {
            updated.tags = updateData.tagsConnect.map((tag: any) => {
                const tagName = typeof tag === 'string' ? tag : tag.tag;
                return {
                    __typename: "Tag",
                    id: `tag_${tagName}_123456789`,
                    tag: tagName,
                    created_at: existing.createdAt,
                    bookmarks: 0,
                    translations: [],
                    you: {
                        __typename: "TagYou",
                        isBookmarked: false,
                    },
                };
            });
        }

        storage[id] = updated;
        return updated;
    },

    async delete(id: string): Promise<{ success: boolean }> {
        const storage = getStorage();
        if (storage[id]) {
            delete storage[id];
            return { success: true };
        }
        return { success: false };
    },
};