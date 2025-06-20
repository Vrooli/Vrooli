import { generatePK, generatePublicId, nanoid } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";

interface TagRelationConfig extends RelationConfig {
    withTranslations?: boolean | number;
    translations?: Array<{ language: string; description: string }>;
    parentId?: string;
    withChildren?: boolean | number;
    bookmarkCount?: number;
}

/**
 * Database fixture factory for Tag model
 * Handles hierarchical tags with multi-language support
 */
export class TagDbFactory extends DatabaseFixtureFactory<
    Prisma.Tag,
    Prisma.TagCreateInput,
    Prisma.TagInclude,
    Prisma.TagUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('Tag', prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.tag;
    }

    protected getMinimalData(overrides?: Partial<Prisma.TagCreateInput>): Prisma.TagCreateInput {
        const uniqueTag = `tag_${nanoid(8)}`.toLowerCase();
        
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            tag: uniqueTag,
            bookmarks: 0,
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<Prisma.TagCreateInput>): Prisma.TagCreateInput {
        const uniqueTag = `complete_tag_${nanoid(6)}`.toLowerCase();
        
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            tag: uniqueTag,
            bookmarks: 25,
            translations: {
                create: [
                    {
                        id: generatePK(),
                        language: "en",
                        description: `A comprehensive test tag: ${uniqueTag}`,
                    },
                    {
                        id: generatePK(),
                        language: "es",
                        description: `Una etiqueta de prueba completa: ${uniqueTag}`,
                    },
                ],
            },
            ...overrides,
        };
    }

    protected getDefaultInclude(): Prisma.TagInclude {
        return {
            translations: true,
            parent: {
                select: {
                    id: true,
                    publicId: true,
                    tag: true,
                },
            },
            children: {
                select: {
                    id: true,
                    publicId: true,
                    tag: true,
                    bookmarks: true,
                },
            },
            _count: {
                select: {
                    children: true,
                    translations: true,
                    resourceTags: true,
                    teamTags: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.TagCreateInput,
        config: TagRelationConfig,
        tx: any
    ): Promise<Prisma.TagCreateInput> {
        let data = { ...baseData };

        // Handle translations
        if (config.withTranslations || config.translations) {
            if (config.translations) {
                data.translations = {
                    create: config.translations.map(trans => ({
                        id: generatePK(),
                        ...trans,
                    })),
                };
            } else if (config.withTranslations) {
                const count = typeof config.withTranslations === 'number' ? config.withTranslations : 2;
                const languages = ['en', 'es', 'fr', 'de', 'ja'].slice(0, count);
                data.translations = {
                    create: languages.map(lang => ({
                        id: generatePK(),
                        language: lang,
                        description: `Description for ${data.tag} in ${lang}`,
                    })),
                };
            }
        }

        // Handle parent relationship
        if (config.parentId) {
            data.parent = {
                connect: { id: config.parentId }
            };
        }

        // Handle bookmark count
        if (config.bookmarkCount !== undefined) {
            data.bookmarks = config.bookmarkCount;
        }

        return data;
    }

    protected async checkModelConstraints(record: Prisma.Tag): Promise<string[]> {
        const violations: string[] = [];
        
        // Check tag name uniqueness
        if (record.tag) {
            const duplicate = await this.prisma.tag.findFirst({
                where: { 
                    tag: record.tag,
                    id: { not: record.id },
                },
            });
            if (duplicate) {
                violations.push('Tag name must be unique');
            }
        }

        // Check tag name format
        if (record.tag && !/^[a-z0-9_-]+$/.test(record.tag)) {
            violations.push('Tag name should only contain lowercase letters, numbers, underscores, and hyphens');
        }

        // Check tag name length
        if (record.tag && (record.tag.length < 2 || record.tag.length > 50)) {
            violations.push('Tag name must be between 2 and 50 characters');
        }

        // Check bookmarks count
        if (record.bookmarks < 0) {
            violations.push('Bookmark count cannot be negative');
        }

        // Check for self-parent reference
        if (record.parentId && record.id && record.parentId === record.id) {
            violations.push('Tag cannot be its own parent');
        }

        // Check parent exists
        if (record.parentId) {
            const parent = await this.prisma.tag.findUnique({
                where: { id: record.parentId }
            });
            if (!parent) {
                violations.push('Parent tag does not exist');
            }
        }

        return violations;
    }

    /**
     * Create a popular tag with high bookmark count
     */
    async createPopularTag(overrides?: Partial<Prisma.TagCreateInput>): Promise<Prisma.Tag> {
        return this.createMinimal({
            bookmarks: Math.floor(Math.random() * 500) + 100, // 100-600 bookmarks
            ...overrides,
        });
    }

    /**
     * Create a tag with children (hierarchical structure)
     */
    async createTagWithChildren(
        parentTag: string, 
        childTags: string[]
    ): Promise<{ parent: Prisma.Tag; children: Prisma.Tag[] }> {
        const parent = await this.createMinimal({
            tag: parentTag.toLowerCase().replace(/\s+/g, '-'),
        });

        const children = [];
        for (const childTag of childTags) {
            const child = await this.createMinimal({
                tag: childTag.toLowerCase().replace(/\s+/g, '-'),
                parent: { connect: { id: parent.id } },
            });
            children.push(child);
        }

        return { parent, children };
    }

    /**
     * Create a multilingual tag with translations
     */
    async createMultilingualTag(
        tag: string,
        descriptions: Record<string, string>
    ): Promise<Prisma.Tag> {
        return this.createWithRelations({
            overrides: { tag: tag.toLowerCase().replace(/\s+/g, '-') },
            translations: Object.entries(descriptions).map(([language, description]) => ({
                language,
                description,
            })),
        });
    }

    /**
     * Create a technology tag cloud
     */
    async createTechTagCloud(): Promise<Prisma.Tag[]> {
        const techTags = [
            'javascript', 'typescript', 'python', 'react', 'nodejs',
            'docker', 'kubernetes', 'aws', 'postgresql', 'mongodb'
        ];

        const tags = [];
        for (const tag of techTags) {
            const created = await this.createMinimal({
                tag,
                bookmarks: Math.floor(Math.random() * 200) + 10,
            });
            tags.push(created);
        }

        return tags;
    }

    /**
     * Create nested tag hierarchy (category > subcategory > specific)
     */
    async createNestedHierarchy(): Promise<{ 
        categories: Prisma.Tag[]; 
        subcategories: Prisma.Tag[]; 
        specific: Prisma.Tag[] 
    }> {
        // Create main categories
        const categories = await Promise.all([
            this.createMinimal({ tag: 'programming' }),
            this.createMinimal({ tag: 'design' }),
            this.createMinimal({ tag: 'business' }),
        ]);

        // Create subcategories
        const subcategories = await Promise.all([
            // Programming subcategories
            this.createMinimal({ 
                tag: 'web-development', 
                parent: { connect: { id: categories[0].id } }
            }),
            this.createMinimal({ 
                tag: 'mobile-development', 
                parent: { connect: { id: categories[0].id } }
            }),
            // Design subcategories
            this.createMinimal({ 
                tag: 'ui-design', 
                parent: { connect: { id: categories[1].id } }
            }),
            this.createMinimal({ 
                tag: 'ux-design', 
                parent: { connect: { id: categories[1].id } }
            }),
        ]);

        // Create specific tags
        const specific = await Promise.all([
            // Web development specific
            this.createMinimal({ 
                tag: 'react-js', 
                parent: { connect: { id: subcategories[0].id } }
            }),
            this.createMinimal({ 
                tag: 'vue-js', 
                parent: { connect: { id: subcategories[0].id } }
            }),
            // Mobile development specific
            this.createMinimal({ 
                tag: 'react-native', 
                parent: { connect: { id: subcategories[1].id } }
            }),
            this.createMinimal({ 
                tag: 'flutter', 
                parent: { connect: { id: subcategories[1].id } }
            }),
        ]);

        return { categories, subcategories, specific };
    }

    /**
     * Get invalid data scenarios
     */
    getInvalidScenarios(): Record<string, any> {
        return {
            missingRequired: {
                // Missing required fields
                publicId: generatePublicId(),
                bookmarks: 0,
            },
            invalidTypes: {
                id: "not-a-bigint",
                publicId: 123, // Should be string
                tag: 123, // Should be string
                bookmarks: "not-a-number", // Should be number
            },
            emptyTag: {
                id: generatePK(),
                publicId: generatePublicId(),
                tag: "", // Empty tag name
                bookmarks: 0,
            },
            invalidTagFormat: {
                id: generatePK(),
                publicId: generatePublicId(),
                tag: "Tag With Spaces!", // Invalid characters
                bookmarks: 0,
            },
            negativeBookmarks: {
                id: generatePK(),
                publicId: generatePublicId(),
                tag: "negative-tag",
                bookmarks: -1, // Negative bookmarks
            },
            selfParent: {
                id: generatePK(),
                publicId: generatePublicId(),
                tag: "self-parent-tag",
                bookmarks: 0,
                parent: { connect: { id: generatePK() } }, // Will be same as own ID
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, Prisma.TagCreateInput> {
        return {
            maxLengthTag: {
                ...this.getMinimalData(),
                tag: 'a'.repeat(50), // Maximum length
            },
            unicodeTag: {
                ...this.getMinimalData(),
                tag: '🏷️-emoji-tag-日本語'.toLowerCase(),
            },
            popularTag: {
                ...this.getMinimalData(),
                bookmarks: 1000,
            },
            multiLanguageTag: {
                ...this.getCompleteData(),
                translations: {
                    create: [
                        { id: generatePK(), language: "en", description: "English description" },
                        { id: generatePK(), language: "es", description: "Descripción en español" },
                        { id: generatePK(), language: "fr", description: "Description en français" },
                        { id: generatePK(), language: "de", description: "Deutsche Beschreibung" },
                        { id: generatePK(), language: "ja", description: "日本語の説明" },
                        { id: generatePK(), language: "zh", description: "中文描述" },
                    ],
                },
            },
            specialCharactersTag: {
                ...this.getMinimalData(),
                tag: "tag-with_underscores-and-hyphens",
            },
        };
    }

    protected getCascadeInclude(): any {
        return {
            translations: true,
            children: true,
            resourceTags: true,
            teamTags: true,
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.Tag,
        remainingDepth: number,
        tx: any
    ): Promise<void> {
        // Delete in order of dependencies
        
        // Delete translation records
        if (record.translations?.length) {
            await tx.tagTranslation.deleteMany({
                where: { tagId: record.id },
            });
        }

        // Delete junction table records
        if (record.resourceTags?.length) {
            await tx.resourceTag.deleteMany({
                where: { tagId: record.id },
            });
        }

        if (record.teamTags?.length) {
            await tx.teamTag.deleteMany({
                where: { tagId: record.id },
            });
        }

        // Handle child tags - set their parentId to null instead of deleting
        if (record.children?.length) {
            await tx.tag.updateMany({
                where: { parentId: record.id },
                data: { parentId: null },
            });
        }
    }
}

// Export factory creator function
export const createTagDbFactory = (prisma: PrismaClient) => new TagDbFactory(prisma);