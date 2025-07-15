import { generatePublicId, nanoid } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
    TestScenario,
} from "./types.js";

interface TagRelationConfig extends RelationConfig {
    withCreatedBy?: boolean | string;
    withTranslations?: boolean | Array<{ language: string; description?: string }>;
    withResources?: boolean | number;
    withTeams?: boolean | number;
}

/**
 * Enhanced database fixture factory for Tag model
 * Provides comprehensive testing capabilities for tagging system
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Unique tag enforcement
 * - Multi-language descriptions
 * - Resource and team associations
 * - Bookmark tracking
 * - Predefined test scenarios
 * - Comprehensive validation
 */
export class TagDbFactory extends EnhancedDatabaseFactory<
    tag,
    Prisma.tagCreateInput,
    Prisma.tagInclude,
    Prisma.tagUpdateInput
> {
    protected scenarios: Record<string, TestScenario> = {};
    constructor(prisma: PrismaClient) {
        super("tag", prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.tag;
    }

    /**
     * Get complete test fixtures for Tag model
     */
    protected getFixtures(): DbTestFixtures<Prisma.tagCreateInput, Prisma.tagUpdateInput> {
        return {
            minimal: {
                id: this.generateId(),
                tag: `tag_${nanoid()}`,
            },
            complete: {
                id: this.generateId(),
                tag: `complete_tag_${nanoid()}`,
                bookmarks: 10,
                createdBy: { connect: { id: this.generateId() } },
                translations: {
                    create: [
                        {
                            id: this.generateId(),
                            language: "en",
                            description: "A comprehensive tag with full description",
                        },
                        {
                            id: this.generateId(),
                            language: "es",
                            description: "Una etiqueta completa con descripción completa",
                        },
                    ],
                },
            },
            invalid: {
                missingRequired: {
                    // Missing id and tag
                    createdBy: { connect: { id: this.generateId() } },
                },
                invalidTypes: {
                    id: "not-a-bigint",
                    tag: 123, // Should be string
                    bookmarks: "ten", // Should be number
                    createdById: true, // Should be string
                },
                missingTag: {
                    id: this.generateId(),
                    // Missing tag name
                    createdBy: { connect: { id: this.generateId() } },
                },
                duplicateTag: {
                    id: this.generateId(),
                    tag: "existing_tag", // Assumes this already exists
                    createdBy: { connect: { id: this.generateId() } },
                },
                exceedsTagLength: {
                    id: this.generateId(),
                    tag: "a".repeat(129), // Exceeds 128 character limit
                    createdBy: { connect: { id: this.generateId() } },
                },
                invalidTagFormat: {
                    id: this.generateId(),
                    tag: "tag with spaces", // Tags typically shouldn't have spaces
                    createdBy: { connect: { id: this.generateId() } },
                },
            },
            edgeCases: {
                minimalTag: {
                    id: this.generateId(),
                    tag: "a", // Single character tag
                },
                maxLengthTag: {
                    id: this.generateId(),
                    tag: "a".repeat(128), // Max length tag
                },
                unicodeTag: {
                    id: this.generateId(),
                    tag: "测试标签", // Unicode tag
                },
                specialCharacterTag: {
                    id: this.generateId(),
                    tag: "tag-with-hyphens_and_underscores",
                },
                numericTag: {
                    id: this.generateId(),
                    tag: "2024",
                },
                popularTag: {
                    id: this.generateId(),
                    tag: `popular_${nanoid()}`,
                    bookmarks: 1000,
                    createdBy: { connect: { id: this.generateId() } },
                    translations: {
                        create: [{
                            id: this.generateId(),
                            language: "en",
                            description: "A highly popular tag used by many",
                        }],
                    },
                },
                multiLanguageTag: {
                    id: this.generateId(),
                    tag: `multilang_${nanoid()}`,
                    createdBy: { connect: { id: this.generateId() } },
                    translations: {
                        create: [
                            {
                                id: this.generateId(),
                                language: "en",
                                description: "English description",
                            },
                            {
                                id: this.generateId(),
                                language: "es",
                                description: "Descripción en español",
                            },
                            {
                                id: this.generateId(),
                                language: "fr",
                                description: "Description en français",
                            },
                            {
                                id: this.generateId(),
                                language: "de",
                                description: "Deutsche Beschreibung",
                            },
                            {
                                id: this.generateId(),
                                language: "ja",
                                description: "日本語の説明",
                            },
                        ],
                    },
                },
                orphanedTag: {
                    id: this.generateId(),
                    tag: `orphaned_${nanoid()}`,
                    // No creator, no associations
                },
            },
            updates: {
                minimal: {
                    bookmarks: 5,
                },
                complete: {
                    bookmarks: 20,
                    translations: {
                        create: [{
                            id: this.generateId(),
                            language: "fr",
                            description: "Description mise à jour",
                        }],
                        update: [{
                            where: { 
                                tagId_language: {
                                    tagId: this.generateId(),
                                    language: "en",
                                },
                            },
                            data: {
                                description: "Updated English description with more details",
                            },
                        }],
                    },
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.tagCreateInput>): Prisma.tagCreateInput {
        const uniqueTag = `tag_${nanoid()}`.toLowerCase();
        
        return {
            id: this.generateId(),
            tag: uniqueTag,
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.tagCreateInput>): Prisma.tagCreateInput {
        const uniqueTag = `complete_${nanoid()}`.toLowerCase();
        
        return {
            id: this.generateId(),
            tag: uniqueTag,
            bookmarks: 0,
            createdBy: { connect: { id: this.generateId() } },
            translations: {
                create: [
                    {
                        id: this.generateId(),
                        language: "en",
                        description: "A comprehensive test tag",
                    },
                    {
                        id: this.generateId(),
                        language: "es",
                        description: "Una etiqueta de prueba completa",
                    },
                ],
            },
            ...overrides,
        };
    }

    /**
     * Initialize test scenarios
     */
    protected initializeScenarios(): void {
        this.scenarios = {
            categoryTag: {
                name: "categoryTag",
                description: "Tag for categorization",
                config: {
                    overrides: {
                        tag: `category_${nanoid()}`,
                        translations: {
                            create: [{
                                id: this.generateId(),
                                language: "en",
                                description: "Category tag for organizing content",
                            }],
                        },
                    },
                },
            },
            technologyTag: {
                name: "technologyTag",
                description: "Technology-related tag",
                config: {
                    overrides: {
                        tag: `tech_${nanoid()}`,
                        translations: {
                            create: [{
                                id: this.generateId(),
                                language: "en",
                                description: "Technology and programming related",
                            }],
                        },
                    },
                },
            },
            languageTag: {
                name: "languageTag",
                description: "Programming language tag",
                config: {
                    overrides: {
                        tag: "typescript",
                        translations: {
                            create: [{
                                id: this.generateId(),
                                language: "en",
                                description: "TypeScript programming language",
                            }],
                        },
                    },
                },
            },
            topicTag: {
                name: "topicTag",
                description: "General topic tag",
                config: {
                    overrides: {
                        tag: `topic_${nanoid()}`,
                        bookmarks: 50,
                        translations: {
                            create: [{
                                id: this.generateId(),
                                language: "en",
                                description: "General topic for discussion",
                            }],
                        },
                    },
                },
            },
            trendingTag: {
                name: "trendingTag",
                description: "Trending/popular tag",
                config: {
                    overrides: {
                        tag: `trending_${nanoid()}`,
                        bookmarks: 500,
                        translations: {
                            create: [{
                                id: this.generateId(),
                                language: "en",
                                description: "Currently trending topic",
                            }],
                        },
                    },
                },
            },
        };
    }

    /**
     * Create tags for specific contexts
     */
    async createCategoryTag(name: string, description?: string): Promise<Prisma.tag> {
        return await this.createMinimal({
            tag: name.toLowerCase().replace(/\s+/g, "-"),
            translations: description ? {
                create: [{
                    id: this.generateId(),
                    language: "en",
                    description,
                }],
            } : undefined,
        });
    }

    async createTechnologyTag(tech: string): Promise<Prisma.tag> {
        return await this.createMinimal({
            tag: tech.toLowerCase(),
            translations: {
                create: [{
                    id: this.generateId(),
                    language: "en",
                    description: `${tech} technology`,
                }],
            },
        });
    }

    async createLanguageTag(language: string): Promise<Prisma.tag> {
        return await this.createMinimal({
            tag: language.toLowerCase(),
            translations: {
                create: [{
                    id: this.generateId(),
                    language: "en",
                    description: `${language} programming language`,
                }],
            },
        });
    }

    /**
     * Create multiple related tags
     */
    async createTagSet(baseName: string, count = 5): Promise<Prisma.tag[]> {
        const tags: Prisma.tag[] = [];
        
        for (let i = 0; i < count; i++) {
            const tag = await this.createMinimal({
                tag: `${baseName}_${i + 1}`.toLowerCase(),
                translations: {
                    create: [{
                        id: this.generateId(),
                        language: "en",
                        description: `${baseName} category ${i + 1}`,
                    }],
                },
            });
            tags.push(tag);
        }
        
        return tags;
    }

    /**
     * Create common tags
     */
    async createCommonTags(): Promise<Record<string, Prisma.tag>> {
        const tags: Record<string, Prisma.tag> = {};
        
        // Programming languages
        tags.javascript = await this.createLanguageTag("javascript");
        tags.typescript = await this.createLanguageTag("typescript");
        tags.python = await this.createLanguageTag("python");
        
        // Categories
        tags.tutorial = await this.createCategoryTag("tutorial", "Educational tutorials and guides");
        tags.documentation = await this.createCategoryTag("documentation", "Documentation and references");
        tags.opensource = await this.createCategoryTag("open-source", "Open source projects and contributions");
        
        // Topics
        tags.webdev = await this.createCategoryTag("web-development", "Web development related");
        tags.mobile = await this.createCategoryTag("mobile", "Mobile development");
        tags.ai = await this.createCategoryTag("artificial-intelligence", "AI and machine learning");
        
        return tags;
    }

    protected getDefaultInclude(): Prisma.tagInclude {
        return {
            createdBy: true,
            translations: true,
            resources: {
                take: 10,
                include: {
                    resource: true,
                },
            },
            teams: {
                take: 10,
                include: {
                    team: true,
                },
            },
            _count: {
                select: {
                    resources: true,
                    teams: true,
                    bookmarkedBy: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.tagCreateInput,
        config: TagRelationConfig,
        tx: any,
    ): Promise<Prisma.tagCreateInput> {
        const data = { ...baseData };

        // Handle createdBy relationship
        if (config.withCreatedBy) {
            const createdById = typeof config.withCreatedBy === "string" ? config.withCreatedBy : this.generateId();
            data.createdById = createdById;
        }

        // Handle translations
        if (config.withTranslations && Array.isArray(config.withTranslations)) {
            data.translations = {
                create: config.withTranslations.map(trans => ({
                    id: this.generateId(),
                    ...trans,
                })),
            };
        }

        // Resource and team associations would be handled separately
        // as they are many-to-many relationships through join tables

        return data;
    }

    protected async checkModelConstraints(record: Prisma.tag): Promise<string[]> {
        const violations: string[] = [];
        
        // Check tag length
        if (record.tag && record.tag.length > 128) {
            violations.push("Tag exceeds maximum length of 128 characters");
        }

        // Check tag uniqueness
        if (record.tag) {
            const duplicate = await this.prisma.tag.findFirst({
                where: {
                    tag: record.tag,
                    id: { not: record.id },
                },
            });
            
            if (duplicate) {
                violations.push("Tag name must be unique");
            }
        }

        // Check tag format (optional - depending on business rules)
        if (record.tag) {
            // Example: tags should be lowercase, no spaces, alphanumeric with hyphens/underscores
            const validFormat = /^[a-z0-9_-]+$/.test(record.tag);
            if (!validFormat) {
                violations.push("Tag should be lowercase alphanumeric with hyphens or underscores only");
            }
        }

        // Check bookmark count is not negative
        if (record.bookmarks < 0) {
            violations.push("Bookmark count cannot be negative");
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {
            translations: true,
            resources: true,
            teams: true,
            bookmarkedBy: true,
            reports: true,
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.tag,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[],
    ): Promise<void> {
        // Helper to check if a relation should be deleted
        const shouldDelete = (relation: string) => 
            !includeOnly || includeOnly.includes(relation);

        // Delete translations
        if (shouldDelete("translations") && record.translations?.length) {
            await tx.tag_translation.deleteMany({
                where: { tagId: record.id },
            });
        }

        // Delete resource associations
        if (shouldDelete("resources") && record.resources?.length) {
            await tx.resource_tag.deleteMany({
                where: { tagId: record.id },
            });
        }

        // Delete team associations
        if (shouldDelete("teams") && record.teams?.length) {
            await tx.team_tag.deleteMany({
                where: { tagId: record.id },
            });
        }

        // Delete bookmarks
        if (shouldDelete("bookmarkedBy") && record.bookmarkedBy?.length) {
            await tx.bookmark.deleteMany({
                where: { tagId: record.id },
            });
        }

        // Delete reports
        if (shouldDelete("reports") && record.reports?.length) {
            await tx.report.deleteMany({
                where: { tagId: record.id },
            });
        }
    }

    /**
     * Find or create a tag
     */
    async findOrCreate(tagName: string, createdById?: string): Promise<Prisma.tag> {
        const normalizedTag = tagName.toLowerCase().trim();
        
        // Try to find existing tag
        const existing = await this.prisma.tag.findUnique({
            where: { tag: normalizedTag },
            include: this.getDefaultInclude(),
        });
        
        if (existing) {
            return existing;
        }
        
        // Create new tag
        return await this.createMinimal({
            tag: normalizedTag,
            createdById,
        });
    }

    /**
     * Search tags by prefix
     */
    async searchTags(prefix: string, limit = 10): Promise<Prisma.tag[]> {
        return await this.prisma.tag.findMany({
            where: {
                tag: {
                    startsWith: prefix.toLowerCase(),
                },
            },
            include: {
                translations: true,
                _count: {
                    select: {
                        resources: true,
                        teams: true,
                        bookmarkedBy: true,
                    },
                },
            },
            orderBy: [
                { bookmarks: "desc" },
                { tag: "asc" },
            ],
            take: limit,
        });
    }

    /**
     * Get popular tags
     */
    async getPopularTags(limit = 20): Promise<Prisma.tag[]> {
        return await this.prisma.tag.findMany({
            where: {
                bookmarks: { gt: 0 },
            },
            include: this.getDefaultInclude(),
            orderBy: { bookmarks: "desc" },
            take: limit,
        });
    }

    /**
     * Associate tag with resources
     */
    async addToResources(tagId: string, resourceIds: string[]): Promise<number> {
        const data = resourceIds.map(resourceId => ({
            id: this.generateId(),
            tagId,
            resourceId,
        }));
        
        const result = await this.prisma.resource_tag.createMany({
            data,
            skipDuplicates: true,
        });
        
        return result.count;
    }

    /**
     * Associate tag with teams
     */
    async addToTeams(tagId: string, teamIds: string[]): Promise<number> {
        const data = teamIds.map(teamId => ({
            id: this.generateId(),
            tagId,
            teamId,
        }));
        
        const result = await this.prisma.team_tag.createMany({
            data,
            skipDuplicates: true,
        });
        
        return result.count;
    }

    /**
     * Increment bookmark count
     */
    async incrementBookmarks(tagId: string): Promise<Prisma.tag> {
        return await this.prisma.tag.update({
            where: { id: tagId },
            data: {
                bookmarks: { increment: 1 },
            },
            include: this.getDefaultInclude(),
        });
    }

    /**
     * Decrement bookmark count
     */
    async decrementBookmarks(tagId: string): Promise<Prisma.tag> {
        return await this.prisma.tag.update({
            where: { id: tagId },
            data: {
                bookmarks: { decrement: 1 },
            },
            include: this.getDefaultInclude(),
        });
    }
}

// Export factory creator function
export const createTagDbFactory = (prisma: PrismaClient) => 
    new TagDbFactory(prisma);

// Export the class for type usage
export { TagDbFactory as TagDbFactoryClass };
