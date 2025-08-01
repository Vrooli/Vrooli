import { type Prisma } from "@prisma/client";
import { generatePK, generatePublicId, PullRequestStatus } from "@vrooli/shared";

/**
 * Database fixtures for PullRequest model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing - using lazy initialization to avoid module-level generatePK() calls
let _pullRequestDbIds: Record<string, bigint> | null = null;
export function getPullRequestDbIds() {
    if (!_pullRequestDbIds) {
        _pullRequestDbIds = {
            pr1: generatePK(),
            pr2: generatePK(),
            pr3: generatePK(),
            pr4: generatePK(),
            pr5: generatePK(),
            translation1: generatePK(),
            translation2: generatePK(),
            translation3: generatePK(),
            translation4: generatePK(),
            translation5: generatePK(),
        };
    }
    return _pullRequestDbIds;
}

/**
 * Minimal pull request data for database creation
 */
export const minimalPullRequestDb: Prisma.pull_requestCreateInput = {
    id: getPullRequestDbIds().pr1,
    publicId: generatePublicId(),
    status: PullRequestStatus.Open,
    translations: {
        create: [{
            id: getPullRequestDbIds().translation1,
            language: "en",
            text: "Simple pull request for changes",
        }],
    },
};

/**
 * Draft pull request with basic fields
 */
export const draftPullRequestDb: Prisma.pull_requestCreateInput = {
    id: getPullRequestDbIds().pr2,
    publicId: generatePublicId(),
    status: PullRequestStatus.Draft,
    translations: {
        create: [{
            id: getPullRequestDbIds().translation2,
            language: "en",
            text: "Work in progress - implementing new feature",
        }],
    },
};

/**
 * Complete pull request with all relationships
 */
export const completePullRequestDb: Prisma.pull_requestCreateInput = {
    id: getPullRequestDbIds().pr3,
    publicId: generatePublicId(),
    status: PullRequestStatus.Open,
    createdAt: new Date("2024-01-15T10:00:00Z"),
    updatedAt: new Date("2024-01-15T14:30:00Z"),
    translations: {
        create: [
            {
                id: getPullRequestDbIds().translation3,
                language: "en",
                text: "Feature implementation: Add support for advanced search filters. This PR includes comprehensive tests and documentation updates.",
            },
            {
                id: getPullRequestDbIds().translation4,
                language: "es",
                text: "Implementación de funciones: Agregar soporte para filtros de búsqueda avanzados. Este PR incluye pruebas completas y actualizaciones de documentación.",
            },
        ],
    },
};

/**
 * Merged pull request
 */
export const mergedPullRequestDb: Prisma.pull_requestCreateInput = {
    id: getPullRequestDbIds().pr4,
    publicId: generatePublicId(),
    status: PullRequestStatus.Merged,
    createdAt: new Date("2024-01-10T09:00:00Z"),
    updatedAt: new Date("2024-01-12T16:45:00Z"),
    closedAt: new Date("2024-01-12T16:45:00Z"),
    translations: {
        create: [{
            id: getPullRequestDbIds().translation5,
            language: "en",
            text: "Bug fix: Resolved issue with data validation in forms",
        }],
    },
};

/**
 * Factory for creating pull request database fixtures with overrides
 */
export class PullRequestDbFactory {
    static createMinimal(overrides?: Partial<Prisma.pull_requestCreateInput>): Prisma.pull_requestCreateInput {
        return {
            ...minimalPullRequestDb,
            id: generatePK(),
            publicId: generatePublicId(),
            ...overrides,
        };
    }

    static createDraft(overrides?: Partial<Prisma.pull_requestCreateInput>): Prisma.pull_requestCreateInput {
        return {
            ...draftPullRequestDb,
            id: generatePK(),
            publicId: generatePublicId(),
            ...overrides,
        };
    }

    static createComplete(overrides?: Partial<Prisma.pull_requestCreateInput>): Prisma.pull_requestCreateInput {
        return {
            ...completePullRequestDb,
            id: generatePK(),
            publicId: generatePublicId(),
            ...overrides,
        };
    }

    static createMerged(overrides?: Partial<Prisma.pull_requestCreateInput>): Prisma.pull_requestCreateInput {
        const closedAt = overrides?.closedAt || new Date();
        return {
            ...mergedPullRequestDb,
            id: generatePK(),
            publicId: generatePublicId(),
            closedAt,
            ...overrides,
        };
    }

    /**
     * Create pull request with specific status
     */
    static createWithStatus(
        status: PullRequestStatus,
        overrides?: Partial<Prisma.pull_requestCreateInput>,
    ): Prisma.pull_requestCreateInput {
        const baseData = this.createMinimal(overrides);
        const closedStatuses = [PullRequestStatus.Merged, PullRequestStatus.Rejected, PullRequestStatus.Canceled];

        return {
            ...baseData,
            status,
            closedAt: closedStatuses.includes(status) ? new Date() : null,
        };
    }

    /**
     * Create pull request with user connection
     */
    static createWithUser(
        userId: string | bigint,
        overrides?: Partial<Prisma.pull_requestCreateInput>,
    ): Prisma.pull_requestCreateInput {
        return {
            ...this.createMinimal(overrides),
            createdBy: { connect: { id: typeof userId === "string" ? BigInt(userId) : userId } },
        };
    }

    /**
     * Create pull request with resource connections
     */
    static createWithResources(
        fromResourceVersionId: string | bigint,
        toResourceId: string | bigint,
        overrides?: Partial<Prisma.pull_requestCreateInput>,
    ): Prisma.pull_requestCreateInput {
        return {
            ...this.createComplete(overrides),
            fromResourceVersion: { connect: { id: typeof fromResourceVersionId === "string" ? BigInt(fromResourceVersionId) : fromResourceVersionId } },
            toResource: { connect: { id: typeof toResourceId === "string" ? BigInt(toResourceId) : toResourceId } },
        };
    }

    /**
     * Create pull request with comments
     */
    static createWithComments(
        commentCount = 2,
        userId: string | bigint,
        overrides?: Partial<Prisma.pull_requestCreateInput>,
    ): Prisma.pull_requestCreateInput {
        const comments = Array.from({ length: commentCount }, (_, i) => ({
            id: generatePK(),
            publicId: generatePublicId(),
            createdBy: { connect: { id: typeof userId === "string" ? BigInt(userId) : userId } },
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    text: `Test comment ${i + 1} on pull request`,
                }],
            },
        }));

        return {
            ...this.createComplete(overrides),
            createdBy: { connect: { id: typeof userId === "string" ? BigInt(userId) : userId } },
            comments: { create: comments },
        };
    }

    /**
     * Create multiple pull requests with different statuses
     */
    static createBatch(
        count: number,
        options?: {
            userId?: string | bigint;
            toResourceId?: string | bigint;
            statuses?: PullRequestStatus[];
        },
    ): Prisma.pull_requestCreateInput[] {
        const statuses = options?.statuses || [
            PullRequestStatus.Open,
            PullRequestStatus.Draft,
            PullRequestStatus.Merged,
        ];

        return Array.from({ length: count }, (_, i) => {
            const status = statuses[i % statuses.length];
            let data = this.createWithStatus(status, {
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        text: `Pull request ${i + 1}: ${status} status`,
                    }],
                },
            });

            if (options?.userId) {
                data = {
                    ...data,
                    createdBy: { connect: { id: typeof options.userId === "string" ? BigInt(options.userId) : options.userId } },
                };
            }

            if (options?.toResourceId) {
                data = {
                    ...data,
                    toResource: { connect: { id: typeof options.toResourceId === "string" ? BigInt(options.toResourceId) : options.toResourceId } },
                };
            }

            return data;
        });
    }
}

/**
 * Helper to seed pull requests for testing
 */
export async function seedPullRequests(
    prisma: any,
    options: {
        count?: number;
        userId?: string | bigint;
        fromResourceVersionId?: string | bigint;
        toResourceId?: string | bigint;
        withComments?: boolean;
        statuses?: PullRequestStatus[];
    } = {},
) {
    const pullRequests = [];
    const count = options.count || 3;

    if (options.fromResourceVersionId && options.toResourceId) {
        // Create pull requests with resource connections
        for (let i = 0; i < count; i++) {
            const prData = PullRequestDbFactory.createWithResources(
                options.fromResourceVersionId,
                options.toResourceId,
                {
                    createdBy: options.userId ? { connect: { id: typeof options.userId === "string" ? BigInt(options.userId) : options.userId } } : undefined,
                    status: options.statuses ? options.statuses[i % options.statuses.length] : PullRequestStatus.Open,
                },
            );

            const pr = await prisma.pull_request.create({
                data: prData,
                include: {
                    translations: true,
                    createdBy: true,
                    fromResourceVersion: true,
                    toResource: true,
                    comments: options.withComments,
                },
            });
            pullRequests.push(pr);
        }
    } else {
        // Create standalone pull requests
        const batch = PullRequestDbFactory.createBatch(count, {
            userId: options.userId,
            toResourceId: options.toResourceId,
            statuses: options.statuses,
        });

        for (const prData of batch) {
            const pr = await prisma.pull_request.create({
                data: prData,
                include: {
                    translations: true,
                    createdBy: true,
                    comments: options.withComments,
                },
            });
            pullRequests.push(pr);
        }
    }

    return pullRequests;
}

/**
 * Helper to update resource version with pull request intent
 */
export async function linkPullRequestToResourceVersion(
    prisma: any,
    pullRequestId: string | bigint,
    resourceVersionId: string | bigint,
) {
    return await prisma.resource_version.update({
        where: { id: typeof resourceVersionId === "string" ? BigInt(resourceVersionId) : resourceVersionId },
        data: {
            pullRequestId: typeof pullRequestId === "string" ? BigInt(pullRequestId) : pullRequestId,
            intendToPullRequest: true,
        },
    });
}
