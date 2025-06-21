/**
 * Fixed Tag Database Factory - Pilot implementation of the corrected architecture
 */

import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.fixed.js";
import { getPrismaTypes } from "./PRISMA_TYPE_MAP.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
} from "./types.fixed.js";

// Temporary ID generation to work around import issues
const generatePK = (): bigint => BigInt(Date.now() * 1000 + Math.floor(Math.random() * 1000));

interface TagRelationConfig extends RelationConfig {
    withTranslations?: boolean;
    languages?: string[];
}

/**
 * Enhanced database fixture factory for Tag model
 * Pilot implementation demonstrating the corrected architecture
 */
export class TagDbFactory extends EnhancedDatabaseFactory<
    Prisma.tag,
    Prisma.tagCreateInput,
    Prisma.tagInclude,
    Prisma.tagUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('tag', prisma);
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
                id: generatePK(),
                tag: 'test-tag',
            },
            
            complete: {
                id: generatePK(),
                tag: 'complete-test-tag',
                translations: {
                    create: [
                        {
                            id: generatePK(),
                            language: 'en',
                            description: 'A comprehensive test tag for testing purposes',
                        },
                    ],
                },
            },
            
            variants: {
                withDescription: {
                    id: generatePK(),
                    tag: 'described-tag',
                    translations: {
                        create: [
                            {
                                id: generatePK(),
                                language: 'en',
                                description: 'Tag with detailed description',
                            },
                        ],
                    },
                },
                
                multiLanguage: {
                    id: generatePK(),
                    tag: 'multilang-tag',
                    translations: {
                        create: [
                            {
                                id: generatePK(),
                                language: 'en',
                                description: 'English description',
                            },
                            {
                                id: generatePK(),
                                language: 'es',
                                description: 'Descripción en español',
                            },
                        ],
                    },
                },
                
                shortTag: {
                    id: generatePK(),
                    tag: 'a',
                },
                
                longTag: {
                    id: generatePK(),
                    tag: 'this-is-a-very-long-tag-name-for-testing-boundaries',
                },
            },
            
            invalid: {
                missingRequired: {
                    id: generatePK(),
                    // Missing required 'tag' field
                },
                
                invalidTypes: {
                    id: generatePK(),
                    tag: 123 as any, // Should be string
                },
                
                emptyTag: {
                    id: generatePK(),
                    tag: '',
                },
                
                nullTag: {
                    id: generatePK(),
                    tag: null as any,
                },
            },
            
            edgeCase: {
                specialCharacters: {
                    id: generatePK(),
                    tag: 'tag-with-special-chars-!@#$%^&*()',
                },
                
                unicodeTag: {
                    id: generatePK(),
                    tag: 'тег-тест-unicode-🏷️',
                },
                
                numbersOnly: {
                    id: generatePK(),
                    tag: '12345',
                },
                
                duplicateId: {
                    id: generatePK(), // This would be used to test unique constraint violations
                    tag: 'duplicate-test',
                },
            },
            
            update: {
                updateTag: {
                    tag: 'updated-tag-name',
                },
                
                addTranslation: {
                    translations: {
                        create: {
                            id: generatePK(),
                            language: 'fr',
                            description: 'Description française',
                        },
                    },
                },
            },
        };
    }

    /**
     * Create tag with specific name
     */
    async createWithTag(tagName: string, overrides?: Partial<Prisma.tagCreateInput>) {
        const data = {
            ...this.getFixtures().minimal,
            tag: tagName,
            ...overrides,
        };
        
        return this.createMinimal(data);
    }

    /**
     * Create tag with translations
     */
    async createWithTranslations(
        tagName: string, 
        translations: Array<{ language: string; description: string }>,
        overrides?: Partial<Prisma.tagCreateInput>
    ) {
        const data: Prisma.tagCreateInput = {
            ...this.getFixtures().minimal,
            tag: tagName,
            translations: {
                create: translations.map(t => ({
                    id: generatePK(),
                    language: t.language,
                    description: t.description,
                })),
            },
            ...overrides,
        };
        
        return this.createMinimal(data);
    }

    /**
     * Create tag with relationships
     */
    async createWithRelations(config: TagRelationConfig) {
        return this.prisma.$transaction(async (tx) => {
            const baseData: Prisma.tagCreateInput = {
                id: generatePK(),
                tag: `tag-${generatePK().toString().slice(-6)}`,
                ...config.overrides,
            };

            if (config.withTranslations) {
                const languages = config.languages || ['en'];
                baseData.translations = {
                    create: languages.map(lang => ({
                        id: generatePK(),
                        language: lang,
                        description: `Description for ${baseData.tag} in ${lang}`,
                    })),
                };
            }

            return tx.tag.create({
                data: baseData,
                include: {
                    translations: true,
                    _count: {
                        select: {
                            translations: true,
                        },
                    },
                },
            });
        });
    }

    /**
     * Verify tag has expected translations
     */
    async verifyTranslations(tagId: string, expectedLanguages: string[]) {
        const tag = await this.prisma.tag.findUnique({
            where: { id: this.convertId(tagId) },
            include: {
                translations: true,
            },
        });
        
        if (!tag) {
            throw new Error(`Tag ${tagId} not found`);
        }
        
        const actualLanguages = tag.translations.map(t => t.language).sort();
        const expectedSorted = expectedLanguages.sort();
        
        if (JSON.stringify(actualLanguages) !== JSON.stringify(expectedSorted)) {
            throw new Error(
                `Translation languages mismatch: expected ${expectedSorted.join(',')}, got ${actualLanguages.join(',')}`
            );
        }
        
        return tag;
    }

    /**
     * Create multiple tags with unique names
     */
    async createMultipleTags(count: number, prefix = 'tag') {
        const tags = [];
        
        for (let i = 0; i < count; i++) {
            const tag = await this.createWithTag(`${prefix}-${i}`);
            tags.push(tag);
        }
        
        return tags;
    }

    /**
     * Search tags by name pattern
     */
    async findTagsByPattern(pattern: string) {
        return this.prisma.tag.findMany({
            where: {
                tag: {
                    contains: pattern,
                },
            },
            include: {
                translations: true,
            },
        });
    }

    /**
     * Get default include for Tag queries
     */
    protected getDefaultInclude(): Prisma.tagInclude {
        return {
            translations: true,
            _count: {
                select: {
                    translations: true,
                },
            },
        };
    }
}

/**
 * Factory function to create TagDbFactory instance
 */
export function createTagDbFactory(prisma: PrismaClient): TagDbFactory {
    return new TagDbFactory(prisma);
}

// Export type for use in other factories or tests
export type { TagRelationConfig };