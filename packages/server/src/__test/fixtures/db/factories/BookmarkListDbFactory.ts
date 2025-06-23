import { generatePK, generatePublicId, nanoid } from "../idHelpers.js";
import { type bookmark_list, type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";

interface BookmarkListRelationConfig extends RelationConfig {
    withBookmarks?: boolean | number;
    bookmarkObjects?: Array<{ objectId: string | bigint; objectType: string }>;
    userId?: string | bigint;
}

/**
 * Database fixture factory for BookmarkList model
 * Handles organized bookmark collections with privacy settings and translations
 */
export class BookmarkListDbFactory extends DatabaseFixtureFactory<
    bookmark_list,
    Prisma.bookmark_listCreateInput,
    Prisma.bookmark_listInclude,
    Prisma.bookmark_listUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super("bookmark_list", prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.bookmark_list;
    }

    protected getMinimalData(overrides?: Partial<Prisma.bookmark_listCreateInput>): Prisma.bookmark_listCreateInput {
        return {
            id: generatePK(),
            label: `Bookmarks_${nanoid(8)}`,
            user: { connect: { id: generatePK() } }, // Will be overridden by relationship config
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<Prisma.bookmark_listCreateInput>): Prisma.bookmark_listCreateInput {
        return {
            id: generatePK(),
            label: "My Bookmark Collection",
            user: { connect: { id: generatePK() } }, // Will be overridden by relationship config
            ...overrides,
        };
    }

    /**
     * Create a private bookmark list
     */
    async createPrivate(userId: string | bigint, overrides?: Partial<Prisma.bookmark_listCreateInput>): Promise<bookmark_list> {
        const data: Prisma.bookmark_listCreateInput = {
            ...this.getMinimalData(),
            label: "Private Bookmarks",
            user: { connect: { id: BigInt(userId) } },
            ...overrides,
        };
        
        const result = await this.prisma.bookmark_list.create({ data });
        this.trackCreatedId(result.id.toString());
        return result;
    }

    /**
     * Create a themed bookmark list
     */
    async createThemed(
        userId: string | bigint,
        theme: string,
        overrides?: Partial<Prisma.bookmark_listCreateInput>,
    ): Promise<bookmark_list> {
        const data: Prisma.bookmark_listCreateInput = {
            ...this.getMinimalData(),
            label: `${theme} Resources`,
            user: { connect: { id: BigInt(userId) } },
            ...overrides,
        };
        
        const result = await this.prisma.bookmark_list.create({ data });
        this.trackCreatedId(result.id.toString());
        return result;
    }

    protected getDefaultInclude(): Prisma.bookmark_listInclude {
        return {
            user: {
                select: {
                    id: true,
                    publicId: true,
                    name: true,
                    handle: true,
                },
            },
            bookmarks: {
                take: 10,
            },
            _count: {
                select: {
                    bookmarks: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.bookmark_listCreateInput,
        config: BookmarkListRelationConfig,
        tx: any,
    ): Promise<Prisma.bookmark_listCreateInput> {
        const data = { ...baseData };

        // Ensure user is set
        if (!data.user && config.userId) {
            data.user = { connect: { id: BigInt(config.userId) } };
        }

        // Handle bookmarks
        if (config.withBookmarks && config.bookmarkObjects && config.bookmarkObjects.length > 0) {
            const bookmarkCount = typeof config.withBookmarks === "number" 
                ? Math.min(config.withBookmarks, config.bookmarkObjects.length)
                : config.bookmarkObjects.length;
                
            data.bookmarks = {
                create: config.bookmarkObjects.slice(0, bookmarkCount).map(obj => {
                    const bookmarkData: any = {
                        id: generatePK(),
                    };
                    
                    // Add the appropriate object connection
                    const objectTypeToField: Record<string, string> = {
                        "resource": "resource",
                        "user": "user",
                        "team": "team",
                        "comment": "comment",
                        "issue": "issue",
                        "tag": "tag",
                    };
                    
                    const fieldName = objectTypeToField[obj.objectType];
                    if (fieldName) {
                        bookmarkData[fieldName] = { connect: { id: BigInt(obj.objectId) } };
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
    async createEmptyList(userId: string | bigint): Promise<bookmark_list> {
        return this.createWithRelations({
            userId,
            overrides: {
                label: "Empty List",
            },
        });
    }

    async createCuratedList(
        userId: string | bigint,
        topic: string,
        bookmarkCount = 5,
    ): Promise<bookmark_list> {
        // Generate dummy bookmark objects
        const bookmarkObjects = Array.from({ length: bookmarkCount }, (_, i) => ({
            objectId: generatePK(),
            objectType: ["resource", "team", "user"][i % 3],
        }));

        return this.createWithRelations({
            userId,
            withBookmarks: true,
            bookmarkObjects,
            overrides: {
                label: `Best of ${topic}`,
            },
        });
    }

    async createMultilingualList(userId: string | bigint): Promise<bookmark_list> {
        return this.createWithRelations({
            userId,
            overrides: {
                label: "Multilingual Bookmarks",
            },
        });
    }

    protected async checkModelConstraints(record: bookmark_list): Promise<string[]> {
        const violations: string[] = [];
        
        // Check label-userId uniqueness
        if (record.label && record.userId) {
            const duplicate = await this.prisma.bookmark_list.findFirst({
                where: { 
                    label: record.label,
                    userId: record.userId,
                    id: { not: record.id },
                },
            });
            if (duplicate) {
                violations.push("Label must be unique per user");
            }
        }

        // Check user exists
        if (record.userId) {
            const user = await this.prisma.user.findUnique({
                where: { id: record.userId },
            });
            if (!user) {
                violations.push("User must exist");
            }
        }

        return violations;
    }

    /**
     * Get invalid data scenarios
     */
    getInvalidScenarios(): Record<string, any> {
        return {
            missingRequired: {
                // Missing id, label, user
            },
            invalidTypes: {
                id: "not-a-snowflake",
                label: 123, // Should be string
                user: "invalid-user-reference", // Should be connect object
            },
            duplicateLabel: {
                id: generatePK(),
                label: "existing_label", // Assumes this exists for a user
                user: { connect: { id: generatePK() } },
            },
            invalidUser: {
                id: generatePK(),
                label: "Test List",
                user: { connect: { id: BigInt("999999999999999") } }, // Non-existent user
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, Prisma.bookmark_listCreateInput> {
        const userId = generatePK();
        return {
            minimalList: {
                ...this.getMinimalData(),
                user: { connect: { id: userId } },
            },
            maxLengthLabel: {
                ...this.getMinimalData(),
                label: "A".repeat(128), // Max length label
                user: { connect: { id: userId } },
            },
            unicodeLabel: {
                ...this.getMinimalData(),
                label: "Unicode Test 🚀📚💡",
                user: { connect: { id: userId } },
            },
            specialCharLabel: {
                ...this.getMinimalData(),
                label: "List with special chars: ñáéíóú ™®©",
                user: { connect: { id: userId } },
            },
        };
    }

    protected getCascadeInclude(): any {
        return {
            bookmarks: true,
        };
    }

    protected async deleteRelatedRecords(
        record: bookmark_list & {
            bookmarks?: any[];
        },
        remainingDepth: number,
        tx: any,
    ): Promise<void> {
        // Delete bookmarks
        if (record.bookmarks?.length) {
            await tx.bookmark.deleteMany({
                where: { listId: record.id },
            });
        }

    }
}

// Export factory creator function
export const createBookmarkListDbFactory = (prisma: PrismaClient) => new BookmarkListDbFactory(prisma);
