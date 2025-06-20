import { generatePK, generatePublicId, nanoid } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";

interface BookmarkListRelationConfig extends RelationConfig {
    withBookmarks?: boolean | number;
    bookmarkObjects?: Array<{ objectId: string; objectType: string }>;
    translations?: Array<{ language: string; name: string; description?: string }>;
}

/**
 * Database fixture factory for BookmarkList model
 * Handles organized bookmark collections with privacy settings and translations
 */
export class BookmarkListDbFactory extends DatabaseFixtureFactory<
    Prisma.BookmarkList,
    Prisma.BookmarkListCreateInput,
    Prisma.BookmarkListInclude,
    Prisma.BookmarkListUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('BookmarkList', prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.bookmarkList;
    }

    protected getMinimalData(overrides?: Partial<Prisma.BookmarkListCreateInput>): Prisma.BookmarkListCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            isPrivate: false,
            createdBy: { connect: { id: generatePK() } }, // Will be overridden by relationship config
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<Prisma.BookmarkListCreateInput>): Prisma.BookmarkListCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            isPrivate: false,
            createdBy: { connect: { id: generatePK() } }, // Will be overridden by relationship config
            translations: {
                create: [
                    {
                        id: generatePK(),
                        language: "en",
                        name: "My Bookmark Collection",
                        description: "A curated collection of my favorite resources and content",
                    },
                    {
                        id: generatePK(),
                        language: "es",
                        name: "Mi Colección de Marcadores",
                        description: "Una colección curada de mis recursos y contenido favoritos",
                    },
                ],
            },
            ...overrides,
        };
    }

    /**
     * Create a private bookmark list
     */
    async createPrivate(createdById: string, overrides?: Partial<Prisma.BookmarkListCreateInput>): Promise<Prisma.BookmarkList> {
        const data: Prisma.BookmarkListCreateInput = {
            ...this.getMinimalData(),
            isPrivate: true,
            createdBy: { connect: { id: createdById } },
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: "Private Bookmarks",
                    description: "My private bookmark collection",
                }],
            },
            ...overrides,
        };
        
        const result = await this.prisma.bookmarkList.create({ data });
        this.trackCreatedId(result.id);
        return result;
    }

    /**
     * Create a themed bookmark list
     */
    async createThemed(
        createdById: string,
        theme: string,
        overrides?: Partial<Prisma.BookmarkListCreateInput>
    ): Promise<Prisma.BookmarkList> {
        const data: Prisma.BookmarkListCreateInput = {
            ...this.getMinimalData(),
            createdBy: { connect: { id: createdById } },
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: `${theme} Resources`,
                    description: `Curated collection of ${theme.toLowerCase()} related content`,
                }],
            },
            ...overrides,
        };
        
        const result = await this.prisma.bookmarkList.create({ data });
        this.trackCreatedId(result.id);
        return result;
    }

    protected getDefaultInclude(): Prisma.BookmarkListInclude {
        return {
            translations: true,
            createdBy: {
                select: {
                    id: true,
                    publicId: true,
                    name: true,
                    handle: true,
                },
            },
            bookmarks: {
                take: 10,
                include: {
                    by: {
                        select: {
                            id: true,
                            name: true,
                            handle: true,
                        },
                    },
                },
            },
            _count: {
                select: {
                    bookmarks: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.BookmarkListCreateInput,
        config: BookmarkListRelationConfig,
        tx: any
    ): Promise<Prisma.BookmarkListCreateInput> {
        let data = { ...baseData };

        // Ensure creator is set
        if (!data.createdBy && config.createdById) {
            data.createdBy = { connect: { id: config.createdById } };
        }

        // Handle translations
        if (config.translations && Array.isArray(config.translations)) {
            data.translations = {
                create: config.translations.map(trans => ({
                    id: generatePK(),
                    ...trans,
                })),
            };
        }

        // Handle bookmarks
        if (config.withBookmarks && config.bookmarkObjects && config.bookmarkObjects.length > 0) {
            const bookmarkCount = typeof config.withBookmarks === 'number' 
                ? Math.min(config.withBookmarks, config.bookmarkObjects.length)
                : config.bookmarkObjects.length;
                
            data.bookmarks = {
                create: config.bookmarkObjects.slice(0, bookmarkCount).map(obj => {
                    const bookmarkData: any = {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        by: data.createdBy, // Same user who created the list
                    };
                    
                    // Add the appropriate object connection
                    const objectTypeToField: Record<string, string> = {
                        'project': 'project',
                        'routine': 'routine',
                        'user': 'user',
                        'team': 'team',
                        'comment': 'comment',
                        'issue': 'issue',
                        'note': 'note',
                        'prompt': 'prompt',
                        'question': 'question',
                        'smartContract': 'smartContract',
                        'standard': 'standard',
                    };
                    
                    const fieldName = objectTypeToField[obj.objectType];
                    if (fieldName) {
                        bookmarkData[fieldName] = { connect: { id: obj.objectId } };
                    }
                    
                    return bookmarkData;
                }),
            };
        }

        return data;
    }

    /**
     * Create test scenarios
     */
    async createEmptyList(createdById: string): Promise<Prisma.BookmarkList> {
        return this.createWithRelations({
            createdById,
            translations: [{
                language: "en",
                name: "Empty List",
                description: "A list with no bookmarks yet",
            }],
        });
    }

    async createCuratedList(
        createdById: string,
        topic: string,
        bookmarkCount: number = 5
    ): Promise<Prisma.BookmarkList> {
        // Generate dummy bookmark objects
        const bookmarkObjects = Array.from({ length: bookmarkCount }, (_, i) => ({
            objectId: generatePK(),
            objectType: ['project', 'routine', 'standard'][i % 3],
        }));

        return this.createWithRelations({
            createdById,
            withBookmarks: true,
            bookmarkObjects,
            translations: [{
                language: "en",
                name: `Best of ${topic}`,
                description: `Curated collection of top ${topic} resources`,
            }],
        });
    }

    async createMultilingualList(createdById: string): Promise<Prisma.BookmarkList> {
        return this.createWithRelations({
            createdById,
            translations: [
                { language: "en", name: "Multilingual Bookmarks", description: "Bookmarks in multiple languages" },
                { language: "es", name: "Marcadores Multilingües", description: "Marcadores en múltiples idiomas" },
                { language: "fr", name: "Signets Multilingues", description: "Signets en plusieurs langues" },
                { language: "de", name: "Mehrsprachige Lesezeichen", description: "Lesezeichen in mehreren Sprachen" },
                { language: "ja", name: "多言語ブックマーク", description: "複数の言語のブックマーク" },
            ],
        });
    }

    protected async checkModelConstraints(record: Prisma.BookmarkList): Promise<string[]> {
        const violations: string[] = [];
        
        // Check publicId uniqueness
        if (record.publicId) {
            const duplicate = await this.prisma.bookmarkList.findFirst({
                where: { 
                    publicId: record.publicId,
                    id: { not: record.id },
                },
            });
            if (duplicate) {
                violations.push('PublicId must be unique');
            }
        }

        // Check creator exists
        if (record.createdById) {
            const creator = await this.prisma.user.findUnique({
                where: { id: record.createdById },
            });
            if (!creator) {
                violations.push('Creator user must exist');
            }
        }

        // Check has at least one translation
        const translations = await this.prisma.bookmarkListTranslation.findMany({
            where: { bookmarkListId: record.id },
        });
        
        if (translations.length === 0) {
            violations.push('BookmarkList should have at least one translation');
        }

        return violations;
    }

    /**
     * Get invalid data scenarios
     */
    getInvalidScenarios(): Record<string, any> {
        return {
            missingRequired: {
                // Missing id, publicId, createdBy
                isPrivate: false,
            },
            invalidTypes: {
                id: "not-a-snowflake",
                publicId: 123, // Should be string
                isPrivate: "yes", // Should be boolean
                createdBy: "invalid-user-reference", // Should be connect object
            },
            duplicatePublicId: {
                id: generatePK(),
                publicId: "existing_public_id", // Assumes this exists
                isPrivate: false,
                createdBy: { connect: { id: generatePK() } },
            },
            invalidCreator: {
                id: generatePK(),
                publicId: generatePublicId(),
                isPrivate: false,
                createdBy: { connect: { id: "999999999999999" } }, // Non-existent user
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, Prisma.BookmarkListCreateInput> {
        const userId = generatePK();
        return {
            minimalPrivateList: {
                ...this.getMinimalData(),
                isPrivate: true,
                createdBy: { connect: { id: userId } },
            },
            maxTranslations: {
                ...this.getMinimalData(),
                createdBy: { connect: { id: userId } },
                translations: {
                    create: Array.from({ length: 20 }, (_, i) => ({
                        id: generatePK(),
                        language: `lang${i}`,
                        name: `Name in language ${i}`,
                        description: `Description in language ${i}`,
                    })),
                },
            },
            unicodeContent: {
                ...this.getMinimalData(),
                createdBy: { connect: { id: userId } },
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        name: "Unicode Test 🚀📚💡",
                        description: "Testing with emojis and special chars: ñáéíóú ™®©",
                    }],
                },
            },
            veryLongDescription: {
                ...this.getMinimalData(),
                createdBy: { connect: { id: userId } },
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        name: "Long Description List",
                        description: "A".repeat(5000), // Very long description
                    }],
                },
            },
        };
    }

    protected getCascadeInclude(): any {
        return {
            translations: true,
            bookmarks: true,
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.BookmarkList,
        remainingDepth: number,
        tx: any
    ): Promise<void> {
        // Delete bookmarks
        if (record.bookmarks?.length) {
            await tx.bookmark.deleteMany({
                where: { listId: record.id },
            });
        }

        // Delete translations
        if (record.translations?.length) {
            await tx.bookmarkListTranslation.deleteMany({
                where: { bookmarkListId: record.id },
            });
        }
    }
}

// Export factory creator function
export const createBookmarkListDbFactory = (prisma: PrismaClient) => new BookmarkListDbFactory(prisma);