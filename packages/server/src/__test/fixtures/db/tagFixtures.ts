import { generatePK, generatePublicId } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";
import { EnhancedDbFactory } from "./EnhancedDbFactory.js";
import type { DbTestFixtures, BulkSeedOptions, BulkSeedResult, DbErrorScenarios } from "./types.js";

/**
 * Database fixtures for Tag model - used for seeding test data
 * Tags are reusable across many object types and support hierarchical relationships
 */

// Consistent IDs for testing
export const tagDbIds = {
    tag1: generatePK(),
    tag2: generatePK(),
    tag3: generatePK(),
    parentTag1: generatePK(),
    childTag1: generatePK(),
    popularTag1: generatePK(),
    translation1: generatePK(),
    translation2: generatePK(),
    translation3: generatePK(),
};

/**
 * Enhanced test fixtures for Tag model following standard structure
 */
export const tagDbFixtures: DbTestFixtures<Prisma.TagUncheckedCreateInput> = {
    minimal: {
        id: generatePK(),
        publicId: generatePublicId(),
        tag: "test-tag",
        bookmarks: 0,
    },
    complete: {
        id: generatePK(),
        publicId: generatePublicId(),
        tag: "complete-tag",
        bookmarks: 25,
        translations: {
            create: [
                {
                    id: generatePK(),
                    language: "en",
                    description: "A comprehensive test tag",
                },
                {
                    id: generatePK(),
                    language: "es",
                    description: "Una etiqueta de prueba completa",
                },
            ],
        },
    },
    invalid: {
        missingRequired: {
            // Missing required tag field
            publicId: generatePublicId(),
            bookmarks: 0,
        },
        invalidTypes: {
            id: "not-a-valid-snowflake",
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
        duplicateTag: {
            id: generatePK(),
            publicId: generatePublicId(),
            tag: "existing-tag", // Would violate unique constraint
            bookmarks: 0,
        },
    },
    edgeCases: {
        maxLengthTag: {
            id: generatePK(),
            publicId: generatePublicId(),
            tag: "a".repeat(255), // Maximum tag length
            bookmarks: 0,
        },
        specialCharactersTag: {
            id: generatePK(),
            publicId: generatePublicId(),
            tag: "tag-with_special.characters!@#$%",
            bookmarks: 0,
        },
        unicodeTag: {
            id: generatePK(),
            publicId: generatePublicId(),
            tag: "🏷️-emoji-tag-日本語",
            bookmarks: 0,
        },
        popularTag: {
            id: generatePK(),
            publicId: generatePublicId(),
            tag: "popular-tag",
            bookmarks: 1000,
        },
        multiLanguageTag: {
            id: generatePK(),
            publicId: generatePublicId(),
            tag: "multi-lang-tag",
            bookmarks: 50,
            translations: {
                create: [
                    { id: generatePK(), language: "en", description: "English description" },
                    { id: generatePK(), language: "es", description: "Descripción en español" },
                    { id: generatePK(), language: "fr", description: "Description en français" },
                    { id: generatePK(), language: "de", description: "Deutsche Beschreibung" },
                    { id: generatePK(), language: "ja", description: "日本語の説明" },
                ],
            },
        },
        hierarchicalTag: {
            id: generatePK(),
            publicId: generatePublicId(),
            tag: "child-tag",
            bookmarks: 10,
            parentId: tagDbIds.parentTag1,
        },
    },
};

/**
 * Enhanced factory for creating tag database fixtures
 */
export class TagDbFactory extends EnhancedDbFactory<Prisma.TagUncheckedCreateInput> {
    
    /**
     * Get the test fixtures for Tag model
     */
    protected getFixtures(): DbTestFixtures<Prisma.TagUncheckedCreateInput> {
        return tagDbFixtures;
    }

    /**
     * Get Tag-specific error scenarios
     */
    protected getErrorScenarios(): DbErrorScenarios {
        return {
            constraints: {
                uniqueViolation: {
                    id: tagDbIds.tag1, // Duplicate ID
                    publicId: generatePublicId(),
                    tag: "existing-tag", // Duplicate tag name
                    bookmarks: 0,
                },
                foreignKeyViolation: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    tag: "test-tag",
                    bookmarks: 0,
                    parentId: BigInt("9999999999999999"), // Non-existent parent tag
                },
                checkConstraintViolation: {
                    id: generatePK(),
                    publicId: "", // Empty publicId
                    tag: "test-tag",
                    bookmarks: 0,
                },
            },
            validation: {
                requiredFieldMissing: tagDbFixtures.invalid.missingRequired,
                invalidDataType: tagDbFixtures.invalid.invalidTypes,
                outOfRange: {
                    id: generatePK(),
                    publicId: "a".repeat(500), // PublicId too long
                    tag: "a".repeat(256), // Tag name too long
                    bookmarks: -1, // Negative bookmarks
                },
            },
            businessLogic: {
                selfParentReference: {
                    id: tagDbIds.tag2,
                    publicId: generatePublicId(),
                    tag: "self-referencing-tag",
                    bookmarks: 0,
                    parentId: tagDbIds.tag2, // Self-reference
                },
                circularParentReference: {
                    id: tagDbIds.tag3,
                    publicId: generatePublicId(),
                    tag: "circular-reference-tag",
                    bookmarks: 0,
                    parentId: tagDbIds.childTag1, // Child of its own child
                },
            },
        };
    }

    /**
     * Generate fresh identifiers for tags
     */
    protected generateFreshIdentifiers(): Record<string, any> {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            // Tags use 'tag' field instead of handle
        };
    }

    /**
     * Tag-specific validation
     */
    protected validateSpecific(data: Prisma.TagUncheckedCreateInput): { errors: string[]; warnings: string[] } {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Check required fields specific to Tag
        if (!data.tag) {
            errors.push("Tag name is required");
        } else if (data.tag.length === 0) {
            errors.push("Tag name cannot be empty");
        } else if (data.tag.length > 255) {
            errors.push("Tag name exceeds maximum length of 255 characters");
        }

        if (!data.publicId) {
            errors.push("Tag publicId is required");
        }

        if (typeof data.bookmarks === 'number' && data.bookmarks < 0) {
            errors.push("Bookmark count cannot be negative");
        }

        // Check for self-parent reference
        if (data.parentId && data.id && data.parentId === data.id) {
            errors.push("Tag cannot be its own parent");
        }

        // Warn about special characters in tag names
        if (data.tag && /[^\w\s\-_\.]/u.test(data.tag)) {
            warnings.push("Tag contains special characters which may affect searchability");
        }

        return { errors, warnings };
    }

    // Static methods for backward compatibility
    static createMinimal(overrides?: Partial<Prisma.TagUncheckedCreateInput>): Prisma.TagUncheckedCreateInput {
        const factory = new TagDbFactory();
        return factory.createMinimal(overrides);
    }

    static createWithTranslations(
        translations: Array<{ language: string; description: string }>,
        overrides?: Partial<Prisma.TagUncheckedCreateInput>
    ): Prisma.TagUncheckedCreateInput {
        const factory = new TagDbFactory();
        return factory.createMinimal({
            ...overrides,
            translations: {
                create: translations.map(t => ({
                    id: generatePK(),
                    language: t.language,
                    description: t.description,
                })),
            },
        });
    }

    static createPopular(overrides?: Partial<Prisma.TagUncheckedCreateInput>): Prisma.TagUncheckedCreateInput {
        const factory = new TagDbFactory();
        return factory.createMinimal({
            bookmarks: 100,
            ...overrides,
        });
    }

    static createComplete(overrides?: Partial<Prisma.TagUncheckedCreateInput>): Prisma.TagUncheckedCreateInput {
        const factory = new TagDbFactory();
        return factory.createComplete(overrides);
    }

    /**
     * Create hierarchical tags with parent-child relationship
     */
    static createHierarchical(
        parentTag: string,
        childTags: string[],
        overrides?: Partial<Prisma.TagUncheckedCreateInput>
    ): Prisma.TagUncheckedCreateInput[] {
        const factory = new TagDbFactory();
        const parentId = generatePK();
        
        const parent = factory.createMinimal({
            id: parentId,
            tag: parentTag,
            ...overrides,
        });
        
        const children = childTags.map(childTag => 
            factory.createMinimal({
                tag: childTag,
                parentId: BigInt(parentId),
                ...overrides,
            })
        );
        
        return [parent, ...children];
    }

    /**
     * Create tag cloud with varying popularity
     */
    static createTagCloud(
        tags: Array<{ name: string; popularity?: number }>
    ): Prisma.TagUncheckedCreateInput[] {
        const factory = new TagDbFactory();
        return tags.map(t => 
            factory.createMinimal({
                tag: t.name,
                bookmarks: t.popularity || Math.floor(Math.random() * 100),
            })
        );
    }

    /**
     * Create tags with translations for multiple languages
     */
    static createMultilingualTag(
        tag: string,
        descriptions: Record<string, string>,
        overrides?: Partial<Prisma.TagUncheckedCreateInput>
    ): Prisma.TagUncheckedCreateInput {
        const factory = new TagDbFactory();
        return factory.createMinimal({
            tag,
            translations: {
                create: Object.entries(descriptions).map(([language, description]) => ({
                    id: generatePK(),
                    language,
                    description,
                })),
            },
            ...overrides,
        });
    }
}

/**
 * Helper to seed tags for testing
 */
export async function seedTags(
    prisma: any,
    tags: string[],
    options?: {
        withTranslations?: boolean;
        popular?: boolean;
    }
): Promise<BulkSeedResult<any>> {
    const createdTags = [];

    for (const tag of tags) {
        const tagData = options?.withTranslations
            ? TagDbFactory.createWithTranslations(
                [
                    { language: "en", description: `Description for ${tag}` },
                    { language: "es", description: `Descripción para ${tag}` },
                ],
                { 
                    tag,
                    ...(options?.popular && { bookmarks: Math.floor(Math.random() * 200) + 50 }),
                }
            )
            : TagDbFactory.createMinimal({ 
                tag,
                ...(options?.popular && { bookmarks: Math.floor(Math.random() * 200) + 50 }),
            });

        const created = await prisma.tag.create({ data: tagData });
        createdTags.push(created);
    }

    return {
        records: createdTags,
        summary: {
            total: createdTags.length,
            withAuth: 0,
            bots: 0,
            teams: 0,
        },
    };
}

/**
 * Helper to seed hierarchical tag structure
 */
export async function seedTagHierarchy(
    prisma: any,
    structure: Array<{
        parent: string;
        children: string[];
        withTranslations?: boolean;
    }>
): Promise<BulkSeedResult<any>> {
    const allTags = [];

    for (const { parent, children, withTranslations } of structure) {
        const tags = TagDbFactory.createHierarchical(parent, children);
        
        for (const tagData of tags) {
            const finalData = withTranslations
                ? {
                    ...tagData,
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            description: `Description for ${tagData.tag}`,
                        }],
                    },
                }
                : tagData;

            const created = await prisma.tag.create({ data: finalData });
            allTags.push(created);
        }
    }

    return {
        records: allTags,
        summary: {
            total: allTags.length,
            withAuth: 0,
            bots: 0,
            teams: 0,
        },
    };
}

/**
 * Helper to seed popular tags with analytics
 */
export async function seedPopularTags(
    prisma: any,
    count: number = 10,
    options?: {
        minBookmarks?: number;
        maxBookmarks?: number;
        withTranslations?: boolean;
    }
): Promise<BulkSeedResult<any>> {
    const tags = [];
    const { minBookmarks = 50, maxBookmarks = 500, withTranslations = false } = options || {};

    for (let i = 0; i < count; i++) {
        const bookmarks = Math.floor(Math.random() * (maxBookmarks - minBookmarks)) + minBookmarks;
        const tagData = TagDbFactory.createPopular({
            tag: `popular-tag-${i + 1}`,
            bookmarks,
            ...(withTranslations && {
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        description: `Popular tag with ${bookmarks} bookmarks`,
                    }],
                },
            }),
        });

        const created = await prisma.tag.create({ data: tagData });
        tags.push(created);
    }

    return {
        records: tags,
        summary: {
            total: tags.length,
            withAuth: 0,
            bots: 0,
            teams: 0,
        },
    };
}