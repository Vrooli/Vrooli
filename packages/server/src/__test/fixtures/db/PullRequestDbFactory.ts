import { generatePK, generatePublicId, PullRequestStatus } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
} from "./types.js";

interface PullRequestRelationConfig extends RelationConfig {
    withFromVersion?: boolean;
    withToVersion?: boolean;
    withTranslations?: boolean;
    withCreatedBy?: boolean;
    fromVersionId?: string;
    toVersionId?: string;
    createdById?: string;
    status?: PullRequestStatus;
}

/**
 * Enhanced database fixture factory for Pull Request model
 * Provides comprehensive testing capabilities for pull requests
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for different PR statuses
 * - Version relationship management
 * - Translation support
 * - Merge scenarios testing
 * - Predefined test scenarios
 */
export class PullRequestDbFactory extends EnhancedDatabaseFactory<
    Prisma.pull_request,
    Prisma.pull_requestCreateInput,
    Prisma.pull_requestInclude,
    Prisma.pull_requestUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('pull_request', prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.pull_request;
    }

    /**
     * Get complete test fixtures for Pull Request model
     */
    protected getFixtures(): DbTestFixtures<Prisma.pull_requestCreateInput, Prisma.pull_requestUpdateInput> {
        const fromVersionId = generatePK();
        const toVersionId = generatePK();
        const createdById = generatePK();
        
        return {
            minimal: {
                id: generatePK(),
                publicId: generatePublicId(),
                status: PullRequestStatus.Open,
                fromVersion: { connect: { id: fromVersionId } },
                toVersion: { connect: { id: toVersionId } },
                createdBy: { connect: { id: createdById } },
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        title: "Test Pull Request",
                        description: "A test pull request for testing purposes",
                    }],
                },
            },
            
            complete: {
                id: generatePK(),
                publicId: generatePublicId(),
                status: PullRequestStatus.Open,
                mergedBy: null,
                mergedAt: null,
                fromVersion: { connect: { id: fromVersionId } },
                toVersion: { connect: { id: toVersionId } },
                createdBy: { connect: { id: createdById } },
                translations: {
                    create: [
                        {
                            id: generatePK(),
                            language: "en",
                            title: "Complete Pull Request",
                            description: "A complete pull request with full details and multiple translations",
                        },
                        {
                            id: generatePK(),
                            language: "es",
                            title: "Solicitud de Extracción Completa",
                            description: "Una solicitud de extracción completa con detalles completos",
                        },
                    ],
                },
            },
            
            variants: {
                openPR: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    status: PullRequestStatus.Open,
                    fromVersion: { connect: { id: fromVersionId } },
                    toVersion: { connect: { id: toVersionId } },
                    createdBy: { connect: { id: createdById } },
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            title: "Open Pull Request",
                            description: "An open pull request awaiting review",
                        }],
                    },
                },
                
                mergedPR: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    status: PullRequestStatus.Merged,
                    mergedBy: { connect: { id: createdById } },
                    mergedAt: new Date(),
                    fromVersion: { connect: { id: fromVersionId } },
                    toVersion: { connect: { id: toVersionId } },
                    createdBy: { connect: { id: createdById } },
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            title: "Merged Pull Request",
                            description: "A successfully merged pull request",
                        }],
                    },
                },
                
                rejectedPR: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    status: PullRequestStatus.Rejected,
                    fromVersion: { connect: { id: fromVersionId } },
                    toVersion: { connect: { id: toVersionId } },
                    createdBy: { connect: { id: createdById } },
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            title: "Rejected Pull Request",
                            description: "A rejected pull request that was not merged",
                        }],
                    },
                },
                
                draftPR: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    status: PullRequestStatus.Draft,
                    fromVersion: { connect: { id: fromVersionId } },
                    toVersion: { connect: { id: toVersionId } },
                    createdBy: { connect: { id: createdById } },
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            title: "Draft Pull Request",
                            description: "A draft pull request still being worked on",
                        }],
                    },
                },
            },
            
            invalid: {
                missingVersions: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    status: PullRequestStatus.Open,
                    createdBy: { connect: { id: createdById } },
                    // Missing fromVersion and toVersion
                },
                
                missingCreator: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    status: PullRequestStatus.Open,
                    fromVersion: { connect: { id: fromVersionId } },
                    toVersion: { connect: { id: toVersionId } },
                    // Missing createdBy
                },
                
                invalidStatus: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    status: "InvalidStatus" as any,
                    fromVersion: { connect: { id: fromVersionId } },
                    toVersion: { connect: { id: toVersionId } },
                    createdBy: { connect: { id: createdById } },
                },
            },
            
            edgeCase: {
                sameFromToVersion: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    status: PullRequestStatus.Open,
                    fromVersion: { connect: { id: fromVersionId } },
                    toVersion: { connect: { id: fromVersionId } }, // Same version
                    createdBy: { connect: { id: createdById } },
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            title: "Self-referencing PR",
                            description: "A PR where from and to versions are the same",
                        }],
                    },
                },
                
                noTranslations: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    status: PullRequestStatus.Open,
                    fromVersion: { connect: { id: fromVersionId } },
                    toVersion: { connect: { id: toVersionId } },
                    createdBy: { connect: { id: createdById } },
                    // No translations provided
                },
            },
            
            update: {
                approve: {
                    status: PullRequestStatus.Merged,
                    mergedAt: new Date(),
                },
                
                reject: {
                    status: PullRequestStatus.Rejected,
                },
                
                convertToDraft: {
                    status: PullRequestStatus.Draft,
                },
                
                reopenPR: {
                    status: PullRequestStatus.Open,
                    mergedBy: null,
                    mergedAt: null,
                },
            },
        };
    }

    /**
     * Create pull request with specific status
     */
    async createWithStatus(
        status: PullRequestStatus,
        fromVersionId: string,
        toVersionId: string,
        createdById: string,
        overrides?: Partial<Prisma.pull_requestCreateInput>
    ) {
        const data = {
            ...this.getFixtures().minimal,
            status,
            fromVersion: { connect: { id: fromVersionId } },
            toVersion: { connect: { id: toVersionId } },
            createdBy: { connect: { id: createdById } },
            ...overrides,
        };
        
        return this.createMinimal(data);
    }

    /**
     * Create open pull request
     */
    async createOpen(
        fromVersionId: string,
        toVersionId: string,
        createdById: string,
        overrides?: Partial<Prisma.pull_requestCreateInput>
    ) {
        return this.createWithStatus(PullRequestStatus.Open, fromVersionId, toVersionId, createdById, overrides);
    }

    /**
     * Create merged pull request
     */
    async createMerged(
        fromVersionId: string,
        toVersionId: string,
        createdById: string,
        mergedById?: string,
        overrides?: Partial<Prisma.pull_requestCreateInput>
    ) {
        const mergeData = {
            status: PullRequestStatus.Merged,
            mergedBy: mergedById ? { connect: { id: mergedById } } : { connect: { id: createdById } },
            mergedAt: new Date(),
            ...overrides,
        };
        
        return this.createWithStatus(PullRequestStatus.Merged, fromVersionId, toVersionId, createdById, mergeData);
    }

    /**
     * Create pull request with relationships
     */
    async createWithRelations(config: PullRequestRelationConfig) {
        return this.prisma.$transaction(async (tx) => {
            let fromVersionId = config.fromVersionId;
            let toVersionId = config.toVersionId;
            let createdById = config.createdById;

            // Create versions if needed (simplified - in real scenario these would be proper routine/project versions)
            if (!fromVersionId && config.withFromVersion) {
                const routine = await tx.routine.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        isInternal: false,
                        isPrivate: false,
                        createdBy: { connect: { id: generatePK() } },
                        versions: {
                            create: {
                                id: generatePK(),
                                publicId: generatePublicId(),
                                versionLabel: "1.0.0",
                                isPrivate: false,
                                directoryListing: {},
                                translations: {
                                    create: {
                                        id: generatePK(),
                                        language: "en",
                                        title: "Test Routine",
                                        description: "A test routine for PR testing",
                                    },
                                },
                            },
                        },
                    },
                    include: { versions: true },
                });
                fromVersionId = routine.versions[0].id;
            }

            if (!toVersionId && config.withToVersion) {
                const routine = await tx.routine.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        isInternal: false,
                        isPrivate: false,
                        createdBy: { connect: { id: generatePK() } },
                        versions: {
                            create: {
                                id: generatePK(),
                                publicId: generatePublicId(),
                                versionLabel: "1.1.0",
                                isPrivate: false,
                                directoryListing: {},
                                translations: {
                                    create: {
                                        id: generatePK(),
                                        language: "en",
                                        title: "Updated Test Routine",
                                        description: "An updated test routine for PR testing",
                                    },
                                },
                            },
                        },
                    },
                    include: { versions: true },
                });
                toVersionId = routine.versions[0].id;
            }

            // Create user if needed
            if (!createdById && config.withCreatedBy) {
                const user = await tx.user.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        name: "PR Creator",
                        handle: `pr_creator_${generatePK().slice(-6)}`,
                        status: "Unlocked",
                        isBot: false,
                        isBotDepictingPerson: false,
                        isPrivate: false,
                    },
                });
                createdById = user.id;
            }

            if (!fromVersionId || !toVersionId || !createdById) {
                throw new Error("fromVersionId, toVersionId, and createdById must be provided or generated");
            }

            const prData: Prisma.pull_requestCreateInput = {
                id: generatePK(),
                publicId: generatePublicId(),
                status: config.status || PullRequestStatus.Open,
                fromVersion: { connect: { id: fromVersionId } },
                toVersion: { connect: { id: toVersionId } },
                createdBy: { connect: { id: createdById } },
            };

            if (config.withTranslations !== false) {
                prData.translations = {
                    create: {
                        id: generatePK(),
                        language: "en",
                        title: "Test Pull Request",
                        description: "A test pull request created with relationships",
                    },
                };
            }

            return tx.pull_request.create({
                data: prData,
                include: {
                    fromVersion: true,
                    toVersion: true,
                    createdBy: true,
                    mergedBy: true,
                    translations: true,
                },
            });
        });
    }

    /**
     * Merge a pull request
     */
    async mergePullRequest(prId: string, mergedById: string) {
        return this.prisma.pull_request.update({
            where: { id: prId },
            data: {
                status: PullRequestStatus.Merged,
                mergedBy: { connect: { id: mergedById } },
                mergedAt: new Date(),
            },
            include: {
                fromVersion: true,
                toVersion: true,
                createdBy: true,
                mergedBy: true,
                translations: true,
            },
        });
    }

    /**
     * Verify pull request status
     */
    async verifyStatus(prId: string, expectedStatus: PullRequestStatus) {
        const pr = await this.prisma.pull_request.findUnique({
            where: { id: prId },
        });
        
        if (!pr) {
            throw new Error(`Pull request ${prId} not found`);
        }
        
        if (pr.status !== expectedStatus) {
            throw new Error(
                `Status mismatch: expected ${expectedStatus}, got ${pr.status}`
            );
        }
        
        return pr;
    }
}

/**
 * Factory function to create PullRequestDbFactory instance
 */
export function createPullRequestDbFactory(prisma: PrismaClient): PullRequestDbFactory {
    return new PullRequestDbFactory(prisma);
}