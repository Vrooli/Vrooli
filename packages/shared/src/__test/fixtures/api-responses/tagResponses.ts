/* c8 ignore start */
/**
 * Tag API Response Fixtures
 * 
 * Comprehensive fixtures for tag management endpoints including
 * tag creation, categorization, search, and trending functionality.
 */

import type {
    Tag,
    TagCreateInput,
    TagUpdateInput,
    TagTranslation,
} from "../../../api/types.js";
import { BaseAPIResponseFactory } from "./base.js";
import type { MockDataOptions } from "./types.js";
import { generatePK } from "../../../id/index.js";

// Constants
const DEFAULT_COUNT = 10;
const DEFAULT_ERROR_RATE = 0.1;
const DEFAULT_DELAY_MS = 500;
const MAX_TAG_LENGTH = 50;
const MAX_DESCRIPTION_LENGTH = 200;

/**
 * Tag API response factory
 */
export class TagResponseFactory extends BaseAPIResponseFactory<
    Tag,
    TagCreateInput,
    TagUpdateInput
> {
    protected readonly entityName = "tag";

    /**
     * Create mock tag data
     */
    createMockData(options?: MockDataOptions): Tag {
        const scenario = options?.scenario || "minimal";
        const now = new Date().toISOString();
        const tagId = options?.overrides?.id || generatePK().toString();

        const baseTag: Tag = {
            __typename: "Tag",
            id: tagId,
            created_at: now,
            updated_at: now,
            tag: "example-tag",
            bookmarks: 0,
            translations: [{
                __typename: "TagTranslation",
                id: generatePK().toString(),
                language: "en",
                description: "An example tag for testing purposes",
            }],
            translationsCount: 1,
            you: {
                isOwn: false,
            },
        };

        if (scenario === "complete" || scenario === "edge-case") {
            return {
                ...baseTag,
                tag: "comprehensive-programming-tag",
                bookmarks: 847,
                translations: [
                    {
                        __typename: "TagTranslation",
                        id: generatePK().toString(),
                        language: "en",
                        description: "Comprehensive programming tag for advanced development concepts",
                    },
                    {
                        __typename: "TagTranslation",
                        id: generatePK().toString(),
                        language: "es",
                        description: "Etiqueta de programación integral para conceptos de desarrollo avanzado",
                    },
                    {
                        __typename: "TagTranslation",
                        id: generatePK().toString(),
                        language: "fr",
                        description: "Balise de programmation complète pour les concepts de développement avancés",
                    },
                ],
                translationsCount: 3,
                you: {
                    isOwn: scenario === "edge-case",
                },
                ...options?.overrides,
            };
        }

        return {
            ...baseTag,
            ...options?.overrides,
        };
    }

    /**
     * Create tag from input
     */
    createFromInput(input: TagCreateInput): Tag {
        const now = new Date().toISOString();
        const tagId = generatePK().toString();

        return {
            __typename: "Tag",
            id: tagId,
            created_at: now,
            updated_at: now,
            tag: input.tag,
            bookmarks: 0,
            translations: input.translationsCreate?.map(t => ({
                __typename: "TagTranslation" as const,
                id: generatePK().toString(),
                language: t.language,
                description: t.description || null,
            })) || [{
                __typename: "TagTranslation" as const,
                id: generatePK().toString(),
                language: "en",
                description: `Tag for ${input.tag}`,
            }],
            translationsCount: input.translationsCreate?.length || 1,
            you: {
                isOwn: true, // User owns newly created tags
            },
        };
    }

    /**
     * Update tag from input
     */
    updateFromInput(existing: Tag, input: TagUpdateInput): Tag {
        const updates: Partial<Tag> = {
            updated_at: new Date().toISOString(),
        };

        if (input.tag !== undefined) updates.tag = input.tag;

        // Handle translation updates
        if (input.translationsUpdate) {
            updates.translations = existing.translations?.map(translation => {
                const update = input.translationsUpdate?.find(u => u.id === translation.id);
                return update ? { ...translation, ...update } : translation;
            });
        }

        return {
            ...existing,
            ...updates,
        };
    }

    /**
     * Validate create input
     */
    async validateCreateInput(input: TagCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (!input.tag) {
            errors.tag = "Tag name is required";
        } else {
            if (input.tag.length < 2) {
                errors.tag = "Tag name must be at least 2 characters";
            } else if (input.tag.length > MAX_TAG_LENGTH) {
                errors.tag = `Tag name must be ${MAX_TAG_LENGTH} characters or less`;
            } else if (!/^[a-zA-Z0-9-_]+$/.test(input.tag)) {
                errors.tag = "Tag name can only contain letters, numbers, hyphens, and underscores";
            }
        }

        // Validate translations
        if (input.translationsCreate) {
            input.translationsCreate.forEach((translation, index) => {
                if (!translation.language) {
                    errors[`translationsCreate.${index}.language`] = "Language is required";
                }

                if (translation.description && translation.description.length > MAX_DESCRIPTION_LENGTH) {
                    errors[`translationsCreate.${index}.description`] = `Description must be ${MAX_DESCRIPTION_LENGTH} characters or less`;
                }
            });
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Validate update input
     */
    async validateUpdateInput(input: TagUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (input.tag !== undefined) {
            if (!input.tag || input.tag.length < 2) {
                errors.tag = "Tag name must be at least 2 characters";
            } else if (input.tag.length > MAX_TAG_LENGTH) {
                errors.tag = `Tag name must be ${MAX_TAG_LENGTH} characters or less`;
            } else if (!/^[a-zA-Z0-9-_]+$/.test(input.tag)) {
                errors.tag = "Tag name can only contain letters, numbers, hyphens, and underscores";
            }
        }

        // Validate translation updates
        if (input.translationsUpdate) {
            input.translationsUpdate.forEach((translation, index) => {
                if (translation.description !== undefined && translation.description && translation.description.length > MAX_DESCRIPTION_LENGTH) {
                    errors[`translationsUpdate.${index}.description`] = `Description must be ${MAX_DESCRIPTION_LENGTH} characters or less`;
                }
            });
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Create popular programming tags
     */
    createProgrammingTags(): Tag[] {
        const programmingTags = [
            { name: "javascript", bookmarks: 1250, desc: "JavaScript programming language" },
            { name: "typescript", bookmarks: 980, desc: "TypeScript - typed JavaScript" },
            { name: "python", bookmarks: 1400, desc: "Python programming language" },
            { name: "react", bookmarks: 1100, desc: "React JavaScript library" },
            { name: "nodejs", bookmarks: 850, desc: "Node.js runtime environment" },
            { name: "api", bookmarks: 720, desc: "Application Programming Interface" },
            { name: "database", bookmarks: 650, desc: "Database management and design" },
            { name: "web-development", bookmarks: 890, desc: "Web development practices" },
            { name: "mobile", bookmarks: 540, desc: "Mobile app development" },
            { name: "ai", bookmarks: 760, desc: "Artificial Intelligence" },
        ];

        return programmingTags.map(tag => 
            this.createMockData({
                overrides: {
                    tag: tag.name,
                    bookmarks: tag.bookmarks,
                    translations: [{
                        __typename: "TagTranslation" as const,
                        id: generatePK().toString(),
                        language: "en",
                        description: tag.desc,
                    }],
                },
            }),
        );
    }

    /**
     * Create category-based tags
     */
    createCategoryTags(): Tag[] {
        const categories = [
            { name: "tutorial", bookmarks: 450, desc: "Educational and instructional content" },
            { name: "beginner", bookmarks: 380, desc: "Content suitable for beginners" },
            { name: "advanced", bookmarks: 320, desc: "Advanced level content" },
            { name: "productivity", bookmarks: 290, desc: "Tools and tips for productivity" },
            { name: "automation", bookmarks: 410, desc: "Automated processes and workflows" },
            { name: "security", bookmarks: 520, desc: "Security-related content" },
            { name: "testing", bookmarks: 360, desc: "Testing methodologies and tools" },
            { name: "deployment", bookmarks: 340, desc: "Deployment and DevOps content" },
        ];

        return categories.map(category => 
            this.createMockData({
                overrides: {
                    tag: category.name,
                    bookmarks: category.bookmarks,
                    translations: [{
                        __typename: "TagTranslation" as const,
                        id: generatePK().toString(),
                        language: "en",
                        description: category.desc,
                    }],
                },
            }),
        );
    }

    /**
     * Create trending tags (top tags by bookmarks)
     */
    createTrendingTags(): Tag[] {
        return this.createProgrammingTags()
            .sort((a, b) => b.bookmarks - a.bookmarks)
            .slice(0, 5);
    }

    /**
     * Create tags with multiple translations
     */
    createMultilingualTags(): Tag[] {
        const multilingualTags = [
            {
                name: "international",
                translations: [
                    { lang: "en", desc: "International content and collaboration" },
                    { lang: "es", desc: "Contenido y colaboración internacional" },
                    { lang: "fr", desc: "Contenu et collaboration internationaux" },
                    { lang: "de", desc: "Internationale Inhalte und Zusammenarbeit" },
                ],
            },
            {
                name: "open-source",
                translations: [
                    { lang: "en", desc: "Open source software and projects" },
                    { lang: "es", desc: "Software y proyectos de código abierto" },
                    { lang: "fr", desc: "Logiciels et projets open source" },
                ],
            },
        ];

        return multilingualTags.map(tag => 
            this.createMockData({
                overrides: {
                    tag: tag.name,
                    bookmarks: Math.floor(Math.random() * 500) + 100,
                    translations: tag.translations.map(trans => ({
                        __typename: "TagTranslation" as const,
                        id: generatePK().toString(),
                        language: trans.lang,
                        description: trans.desc,
                    })),
                    translationsCount: tag.translations.length,
                },
            }),
        );
    }

    /**
     * Create duplicate tag error response
     */
    createDuplicateTagErrorResponse(existingTagName: string) {
        return this.createBusinessErrorResponse("duplicate", {
            resource: "tag",
            existingTagName,
            message: `Tag '${existingTagName}' already exists`,
        });
    }

    /**
     * Create invalid characters error response
     */
    createInvalidCharactersErrorResponse(invalidChars: string[]) {
        return this.createValidationErrorResponse({
            tag: `Tag name contains invalid characters: ${invalidChars.join(", ")}. Only letters, numbers, hyphens, and underscores are allowed.`,
        });
    }

    /**
     * Create tag too popular error response (for deletion)
     */
    createTagTooPopularErrorResponse(bookmarkCount: number, threshold = 1000) {
        return this.createBusinessErrorResponse("too_popular", {
            resource: "tag",
            bookmarkCount,
            threshold,
            message: `Cannot delete tag with ${bookmarkCount} bookmarks (threshold: ${threshold})`,
        });
    }
}

/**
 * Pre-configured tag response scenarios
 */
export const tagResponseScenarios = {
    // Success scenarios
    createSuccess: (input?: Partial<TagCreateInput>) => {
        const factory = new TagResponseFactory();
        const defaultInput: TagCreateInput = {
            tag: "new-tag",
            translationsCreate: [{
                language: "en",
                description: "A new tag",
            }],
            ...input,
        };
        return factory.createSuccessResponse(
            factory.createFromInput(defaultInput),
        );
    },

    findSuccess: (tag?: Tag) => {
        const factory = new TagResponseFactory();
        return factory.createSuccessResponse(
            tag || factory.createMockData(),
        );
    },

    findCompleteSuccess: () => {
        const factory = new TagResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({ scenario: "complete" }),
        );
    },

    updateSuccess: (existing?: Tag, updates?: Partial<TagUpdateInput>) => {
        const factory = new TagResponseFactory();
        const tag = existing || factory.createMockData({ scenario: "complete" });
        const input: TagUpdateInput = {
            id: tag.id,
            ...updates,
        };
        return factory.createSuccessResponse(
            factory.updateFromInput(tag, input),
        );
    },

    listSuccess: (tags?: Tag[]) => {
        const factory = new TagResponseFactory();
        return factory.createPaginatedResponse(
            tags || Array.from({ length: DEFAULT_COUNT }, () => factory.createMockData()),
            { page: 1, totalCount: tags?.length || DEFAULT_COUNT },
        );
    },

    programmingTagsSuccess: () => {
        const factory = new TagResponseFactory();
        const tags = factory.createProgrammingTags();
        return factory.createPaginatedResponse(
            tags,
            { page: 1, totalCount: tags.length },
        );
    },

    categoryTagsSuccess: () => {
        const factory = new TagResponseFactory();
        const tags = factory.createCategoryTags();
        return factory.createPaginatedResponse(
            tags,
            { page: 1, totalCount: tags.length },
        );
    },

    trendingTagsSuccess: () => {
        const factory = new TagResponseFactory();
        const tags = factory.createTrendingTags();
        return factory.createPaginatedResponse(
            tags,
            { page: 1, totalCount: tags.length },
        );
    },

    multilingualTagsSuccess: () => {
        const factory = new TagResponseFactory();
        const tags = factory.createMultilingualTags();
        return factory.createPaginatedResponse(
            tags,
            { page: 1, totalCount: tags.length },
        );
    },

    searchSuccess: (searchTerm: string) => {
        const factory = new TagResponseFactory();
        const allTags = [
            ...factory.createProgrammingTags(),
            ...factory.createCategoryTags(),
        ];
        const filteredTags = allTags.filter(tag => 
            tag.tag.includes(searchTerm.toLowerCase()) ||
            tag.translations.some(t => t.description?.toLowerCase().includes(searchTerm.toLowerCase())),
        );
        return factory.createPaginatedResponse(
            filteredTags,
            { page: 1, totalCount: filteredTags.length },
        );
    },

    popularTagsSuccess: (minBookmarks = 500) => {
        const factory = new TagResponseFactory();
        const allTags = [
            ...factory.createProgrammingTags(),
            ...factory.createCategoryTags(),
        ];
        const popularTags = allTags.filter(tag => tag.bookmarks >= minBookmarks);
        return factory.createPaginatedResponse(
            popularTags,
            { page: 1, totalCount: popularTags.length },
        );
    },

    // Error scenarios
    createValidationError: () => {
        const factory = new TagResponseFactory();
        return factory.createValidationErrorResponse({
            tag: "Tag name is required and must be unique",
            "translationsCreate.0.language": "Language is required",
        });
    },

    notFoundError: (tagId?: string) => {
        const factory = new TagResponseFactory();
        return factory.createNotFoundErrorResponse(
            tagId || "non-existent-tag",
        );
    },

    permissionError: (operation?: string) => {
        const factory = new TagResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
            ["tag:write"],
        );
    },

    duplicateTagError: (existingTagName = "existing-tag") => {
        const factory = new TagResponseFactory();
        return factory.createDuplicateTagErrorResponse(existingTagName);
    },

    invalidCharactersError: (invalidChars = ["@", "#", "!", "%"]) => {
        const factory = new TagResponseFactory();
        return factory.createInvalidCharactersErrorResponse(invalidChars);
    },

    tagTooShortError: () => {
        const factory = new TagResponseFactory();
        return factory.createValidationErrorResponse({
            tag: "Tag name must be at least 2 characters",
        });
    },

    tagTooLongError: () => {
        const factory = new TagResponseFactory();
        return factory.createValidationErrorResponse({
            tag: `Tag name must be ${MAX_TAG_LENGTH} characters or less`,
        });
    },

    tagTooPopularError: (bookmarkCount = 1500) => {
        const factory = new TagResponseFactory();
        return factory.createTagTooPopularErrorResponse(bookmarkCount);
    },

    // MSW handlers
    handlers: {
        success: () => new TagResponseFactory().createMSWHandlers(),
        withErrors: function createWithErrors(errorRate?: number) {
            return new TagResponseFactory().createMSWHandlers({ errorRate: errorRate ?? DEFAULT_ERROR_RATE });
        },
        withDelay: function createWithDelay(delay?: number) {
            return new TagResponseFactory().createMSWHandlers({ delay: delay ?? DEFAULT_DELAY_MS });
        },
    },
};

// Export factory instance for direct use
export const tagResponseFactory = new TagResponseFactory();
