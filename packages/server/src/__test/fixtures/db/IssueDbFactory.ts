import { generatePK, generatePublicId, nanoid, IssueStatus } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
    TestScenario,
} from "./types.js";

interface IssueRelationConfig extends RelationConfig {
    withResource?: boolean | string;
    withTeam?: boolean | string;
    withCreatedBy?: boolean | string;
    withClosedBy?: boolean | string;
    withTranslations?: boolean | Array<{ language: string; name: string; description?: string }>;
    withComments?: boolean | number;
}

/**
 * Enhanced database fixture factory for Issue model
 * Provides comprehensive testing capabilities for issue tracking system
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Issue status management
 * - Resource and team associations
 * - Multi-language translations
 * - Comment thread support
 * - View and bookmark tracking
 * - Predefined test scenarios
 * - Comprehensive validation
 */
export class IssueDbFactory extends EnhancedDatabaseFactory<
    Prisma.issue,
    Prisma.issueCreateInput,
    Prisma.issueInclude,
    Prisma.issueUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('issue', prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.issue;
    }

    /**
     * Get complete test fixtures for Issue model
     */
    protected getFixtures(): DbTestFixtures<Prisma.issueCreateInput, Prisma.issueUpdateInput> {
        return {
            minimal: {
                id: generatePK().toString(),
                publicId: generatePublicId(),
                status: IssueStatus.Open,
                createdById: generatePK().toString(),
                translations: {
                    create: [{
                        id: generatePK().toString(),
                        language: "en",
                        name: "Test Issue",
                        description: "This is a test issue",
                    }],
                },
            },
            complete: {
                id: generatePK().toString(),
                publicId: generatePublicId(),
                status: IssueStatus.Open,
                score: 10,
                bookmarks: 5,
                views: 25,
                resourceId: generatePK().toString(),
                teamId: generatePK().toString(),
                createdById: generatePK().toString(),
                translations: {
                    create: [
                        {
                            id: generatePK().toString(),
                            language: "en",
                            name: "Complete Test Issue",
                            description: "This is a comprehensive test issue with all fields populated",
                        },
                        {
                            id: generatePK().toString(),
                            language: "es",
                            name: "Problema de Prueba Completo",
                            description: "Este es un problema de prueba completo con todos los campos poblados",
                        },
                    ],
                },
            },
            invalid: {
                missingRequired: {
                    // Missing id, publicId, and createdById
                    status: IssueStatus.Open,
                },
                invalidTypes: {
                    id: "not-a-bigint",
                    publicId: 123, // Should be string
                    status: "open", // Should be IssueStatus enum
                    score: "ten", // Should be number
                    views: true, // Should be number
                },
                missingTranslations: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    status: IssueStatus.Open,
                    createdById: generatePK().toString(),
                    // Missing translations
                },
                invalidPublicId: {
                    id: generatePK().toString(),
                    publicId: "invalid_public_id_too_long", // Should be 12 chars
                    status: IssueStatus.Open,
                    createdById: generatePK().toString(),
                },
                closedWithoutCloser: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    status: IssueStatus.Resolved,
                    closedAt: new Date(),
                    // Missing closedById when status is closed
                    createdById: generatePK().toString(),
                },
            },
            edgeCases: {
                selfClosedIssue: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    status: IssueStatus.Resolved,
                    createdById: generatePK().toString(),
                    closedById: generatePK().toString(), // Same as createdById
                    closedAt: new Date(),
                    translations: {
                        create: [{
                            id: generatePK().toString(),
                            language: "en",
                            name: "Self-resolved Issue",
                            description: "Issue created and closed by the same user",
                        }],
                    },
                },
                reopenedIssue: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    status: IssueStatus.Open,
                    createdById: generatePK().toString(),
                    closedById: generatePK().toString(),
                    closedAt: null, // Was closed but reopened
                    translations: {
                        create: [{
                            id: generatePK().toString(),
                            language: "en",
                            name: "Reopened Issue",
                            description: "This issue was closed but has been reopened",
                        }],
                    },
                },
                highActivityIssue: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    status: IssueStatus.Open,
                    score: 500,
                    bookmarks: 100,
                    views: 10000,
                    createdById: generatePK().toString(),
                    translations: {
                        create: [{
                            id: generatePK().toString(),
                            language: "en",
                            name: "Critical Production Bug",
                            description: "High-priority issue affecting many users",
                        }],
                    },
                },
                maxLengthFields: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    status: IssueStatus.Open,
                    createdById: generatePK().toString(),
                    translations: {
                        create: [{
                            id: generatePK().toString(),
                            language: "en",
                            name: 'a'.repeat(128), // Max length name
                            description: 'b'.repeat(2048), // Max length description
                        }],
                    },
                },
                multiLanguageIssue: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    status: IssueStatus.Open,
                    createdById: generatePK().toString(),
                    translations: {
                        create: [
                            {
                                id: generatePK().toString(),
                                language: "en",
                                name: "Multi-language Support Issue",
                                description: "Issue with translations in multiple languages",
                            },
                            {
                                id: generatePK().toString(),
                                language: "es",
                                name: "Problema de Soporte Multiidioma",
                                description: "Problema con traducciones en múltiples idiomas",
                            },
                            {
                                id: generatePK().toString(),
                                language: "fr",
                                name: "Problème de Support Multilingue",
                                description: "Problème avec des traductions dans plusieurs langues",
                            },
                            {
                                id: generatePK().toString(),
                                language: "de",
                                name: "Mehrsprachiges Support-Problem",
                                description: "Problem mit Übersetzungen in mehreren Sprachen",
                            },
                        ],
                    },
                },
                orphanedIssue: {
                    id: generatePK().toString(),
                    publicId: generatePublicId(),
                    status: IssueStatus.Open,
                    createdById: generatePK().toString(),
                    // No resource or team association
                    translations: {
                        create: [{
                            id: generatePK().toString(),
                            language: "en",
                            name: "Orphaned Issue",
                            description: "Issue not associated with any resource or team",
                        }],
                    },
                },
            },
            updates: {
                minimal: {
                    status: IssueStatus.InProgress,
                },
                complete: {
                    status: IssueStatus.Resolved,
                    closedAt: new Date(),
                    closedById: generatePK().toString(),
                    score: 20,
                    bookmarks: 10,
                    views: 50,
                    translations: {
                        update: [{
                            where: { 
                                issueId_language: {
                                    issueId: generatePK().toString(),
                                    language: "en",
                                },
                            },
                            data: {
                                name: "Updated Issue Title",
                                description: "Updated description with resolution details",
                            },
                        }],
                        create: [{
                            id: generatePK().toString(),
                            language: "ja",
                            name: "更新された問題",
                            description: "解決の詳細を含む更新された説明",
                        }],
                    },
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.issueCreateInput>): Prisma.issueCreateInput {
        return {
            id: generatePK().toString(),
            publicId: generatePublicId(),
            status: IssueStatus.Open,
            createdById: generatePK().toString(),
            translations: {
                create: [{
                    id: generatePK().toString(),
                    language: "en",
                    name: "Test Issue",
                    description: "Test issue description",
                }],
            },
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.issueCreateInput>): Prisma.issueCreateInput {
        return {
            id: generatePK().toString(),
            publicId: generatePublicId(),
            status: IssueStatus.Open,
            score: 0,
            bookmarks: 0,
            views: 0,
            resourceId: generatePK().toString(),
            teamId: generatePK().toString(),
            createdById: generatePK().toString(),
            translations: {
                create: [
                    {
                        id: generatePK().toString(),
                        language: "en",
                        name: "Complete Test Issue",
                        description: "This is a comprehensive test issue with detailed information",
                    },
                    {
                        id: generatePK().toString(),
                        language: "es",
                        name: "Problema de Prueba Completo",
                        description: "Este es un problema de prueba completo con información detallada",
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
            bugReport: {
                name: "bugReport",
                description: "Bug report issue",
                config: {
                    overrides: {
                        status: IssueStatus.Open,
                        translations: {
                            create: [{
                                id: generatePK().toString(),
                                language: "en",
                                name: "Button not responding on mobile",
                                description: "The submit button doesn't work on iOS Safari",
                            }],
                        },
                    },
                },
            },
            featureRequest: {
                name: "featureRequest",
                description: "Feature request issue",
                config: {
                    overrides: {
                        status: IssueStatus.Open,
                        translations: {
                            create: [{
                                id: generatePK().toString(),
                                language: "en",
                                name: "Add dark mode support",
                                description: "It would be great to have a dark mode option",
                            }],
                        },
                    },
                },
            },
            inProgressIssue: {
                name: "inProgressIssue",
                description: "Issue being worked on",
                config: {
                    overrides: {
                        status: IssueStatus.InProgress,
                        translations: {
                            create: [{
                                id: generatePK().toString(),
                                language: "en",
                                name: "Performance optimization",
                                description: "Improving page load times",
                            }],
                        },
                    },
                },
            },
            resolvedIssue: {
                name: "resolvedIssue",
                description: "Resolved issue",
                config: {
                    overrides: {
                        status: IssueStatus.Resolved,
                        closedAt: new Date(),
                        closedById: generatePK().toString(),
                        translations: {
                            create: [{
                                id: generatePK().toString(),
                                language: "en",
                                name: "Fixed login error",
                                description: "Login error has been resolved in v2.1.0",
                            }],
                        },
                    },
                },
            },
            teamIssue: {
                name: "teamIssue",
                description: "Team-managed issue",
                config: {
                    overrides: {
                        teamId: generatePK().toString(),
                        translations: {
                            create: [{
                                id: generatePK().toString(),
                                language: "en",
                                name: "Team workspace improvements",
                                description: "Various improvements for team collaboration",
                            }],
                        },
                    },
                },
            },
        };
    }

    /**
     * Create issues with specific statuses
     */
    async createOpenIssue(createdById: string, title: string, description: string): Promise<Prisma.issue> {
        return await this.createMinimal({
            createdById,
            status: IssueStatus.Open,
            translations: {
                create: [{
                    id: generatePK().toString(),
                    language: "en",
                    name: title,
                    description,
                }],
            },
        });
    }

    async createInProgressIssue(createdById: string, title: string, description: string): Promise<Prisma.issue> {
        return await this.createMinimal({
            createdById,
            status: IssueStatus.InProgress,
            translations: {
                create: [{
                    id: generatePK().toString(),
                    language: "en",
                    name: title,
                    description,
                }],
            },
        });
    }

    async createResolvedIssue(createdById: string, closedById: string, title: string, resolution: string): Promise<Prisma.issue> {
        return await this.createMinimal({
            createdById,
            status: IssueStatus.Resolved,
            closedById,
            closedAt: new Date(),
            translations: {
                create: [{
                    id: generatePK().toString(),
                    language: "en",
                    name: title,
                    description: resolution,
                }],
            },
        });
    }

    /**
     * Create issue for specific contexts
     */
    async createResourceIssue(resourceId: string, createdById: string, title: string, description: string): Promise<Prisma.issue> {
        return await this.createMinimal({
            resourceId,
            createdById,
            translations: {
                create: [{
                    id: generatePK().toString(),
                    language: "en",
                    name: title,
                    description,
                }],
            },
        });
    }

    async createTeamIssue(teamId: string, createdById: string, title: string, description: string): Promise<Prisma.issue> {
        return await this.createMinimal({
            teamId,
            createdById,
            translations: {
                create: [{
                    id: generatePK().toString(),
                    language: "en",
                    name: title,
                    description,
                }],
            },
        });
    }

    /**
     * Update issue status
     */
    async updateStatus(issueId: string, status: IssueStatus, closedById?: string): Promise<Prisma.issue> {
        const updateData: Prisma.issueUpdateInput = { status };
        
        if (status === IssueStatus.Resolved || status === IssueStatus.Rejected) {
            updateData.closedAt = new Date();
            if (closedById) {
                updateData.closedById = closedById;
            }
        } else {
            updateData.closedAt = null;
            updateData.closedBy = { disconnect: true };
        }
        
        return await this.prisma.issue.update({
            where: { id: issueId },
            data: updateData,
            include: this.getDefaultInclude(),
        });
    }

    protected getDefaultInclude(): Prisma.issueInclude {
        return {
            resource: true,
            team: true,
            createdBy: true,
            closedBy: true,
            translations: true,
            comments: {
                take: 10,
                orderBy: { createdAt: 'desc' },
                include: {
                    ownedByUser: true,
                    translations: true,
                },
            },
            _count: {
                select: {
                    comments: true,
                    bookmarkedBy: true,
                    reactions: true,
                    viewedBy: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.issueCreateInput,
        config: IssueRelationConfig,
        tx: any
    ): Promise<Prisma.issueCreateInput> {
        let data = { ...baseData };

        // Handle resource relationship
        if (config.withResource) {
            const resourceId = typeof config.withResource === 'string' ? config.withResource : generatePK().toString();
            data.resourceId = resourceId;
        }

        // Handle team relationship
        if (config.withTeam) {
            const teamId = typeof config.withTeam === 'string' ? config.withTeam : generatePK().toString();
            data.teamId = teamId;
        }

        // Handle createdBy relationship
        if (config.withCreatedBy) {
            const createdById = typeof config.withCreatedBy === 'string' ? config.withCreatedBy : generatePK().toString();
            data.createdById = createdById;
        }

        // Handle closedBy relationship
        if (config.withClosedBy) {
            const closedById = typeof config.withClosedBy === 'string' ? config.withClosedBy : generatePK().toString();
            data.closedById = closedById;
            data.closedAt = new Date();
            if (data.status === IssueStatus.Open) {
                data.status = IssueStatus.Resolved;
            }
        }

        // Handle translations
        if (config.withTranslations && Array.isArray(config.withTranslations)) {
            data.translations = {
                create: config.withTranslations.map(trans => ({
                    id: generatePK().toString(),
                    ...trans,
                })),
            };
        }

        // Comments will be created after the issue is created
        if (config.withComments && typeof config.withComments === 'number') {
            // This will be handled in post-creation
        }

        return data;
    }

    protected async checkModelConstraints(record: Prisma.issue): Promise<string[]> {
        const violations: string[] = [];
        
        // Check publicId format
        if (record.publicId && record.publicId.length !== 12) {
            violations.push('PublicId must be exactly 12 characters');
        }

        // Check status validity
        const validStatuses = Object.values(IssueStatus);
        if (!validStatuses.includes(record.status as IssueStatus)) {
            violations.push('Invalid issue status');
        }

        // Check closed status consistency
        if ((record.status === IssueStatus.Resolved || record.status === IssueStatus.Rejected)) {
            if (!record.closedAt) {
                violations.push('Closed issues must have closedAt timestamp');
            }
            if (!record.closedById) {
                violations.push('Closed issues must have closedById');
            }
        } else {
            if (record.closedAt) {
                violations.push('Open/InProgress issues should not have closedAt timestamp');
            }
        }

        // Check translations exist
        const translations = await this.prisma.issue_translation.findMany({
            where: { issueId: record.id },
        });
        
        if (translations.length === 0) {
            violations.push('Issue must have at least one translation');
        }

        // Check required translation fields
        for (const translation of translations) {
            if (!translation.name || translation.name.trim() === '') {
                violations.push(`Translation for ${translation.language} missing required name`);
            }
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {
            translations: true,
            comments: true,
            reactions: true,
            reactionSummaries: true,
            bookmarkedBy: true,
            viewedBy: true,
            reports: true,
            subscriptions: true,
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.issue,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[]
    ): Promise<void> {
        // Helper to check if a relation should be deleted
        const shouldDelete = (relation: string) => 
            !includeOnly || includeOnly.includes(relation);

        // Delete comments (and their sub-relations)
        if (shouldDelete('comments') && record.comments?.length) {
            for (const comment of record.comments) {
                // Comments have their own cascade delete logic
                await tx.comment.delete({
                    where: { id: comment.id },
                });
            }
        }

        // Delete translations
        if (shouldDelete('translations') && record.translations?.length) {
            await tx.issue_translation.deleteMany({
                where: { issueId: record.id },
            });
        }

        // Delete reactions
        if (shouldDelete('reactions') && record.reactions?.length) {
            await tx.reaction.deleteMany({
                where: { issueId: record.id },
            });
        }

        // Delete reaction summaries
        if (shouldDelete('reactionSummaries') && record.reactionSummaries?.length) {
            await tx.reaction_summary.deleteMany({
                where: { issueId: record.id },
            });
        }

        // Delete bookmarks
        if (shouldDelete('bookmarkedBy') && record.bookmarkedBy?.length) {
            await tx.bookmark.deleteMany({
                where: { issueId: record.id },
            });
        }

        // Delete views
        if (shouldDelete('viewedBy') && record.viewedBy?.length) {
            await tx.view.deleteMany({
                where: { issueId: record.id },
            });
        }

        // Delete reports
        if (shouldDelete('reports') && record.reports?.length) {
            await tx.report.deleteMany({
                where: { issueId: record.id },
            });
        }

        // Delete notification subscriptions
        if (shouldDelete('subscriptions') && record.subscriptions?.length) {
            await tx.notification_subscription.deleteMany({
                where: { issueId: record.id },
            });
        }
    }

    /**
     * Get issues by status
     */
    async getIssuesByStatus(status: IssueStatus, limit: number = 10): Promise<Prisma.issue[]> {
        return await this.prisma.issue.findMany({
            where: { status },
            include: this.getDefaultInclude(),
            orderBy: { updatedAt: 'desc' },
            take: limit,
        });
    }

    /**
     * Get issues for a resource
     */
    async getResourceIssues(resourceId: string, includeResolved: boolean = false): Promise<Prisma.issue[]> {
        const whereClause: Prisma.issueWhereInput = { resourceId };
        
        if (!includeResolved) {
            whereClause.status = { not: IssueStatus.Resolved };
        }
        
        return await this.prisma.issue.findMany({
            where: whereClause,
            include: this.getDefaultInclude(),
            orderBy: { createdAt: 'desc' },
        });
    }

    /**
     * Get issues for a team
     */
    async getTeamIssues(teamId: string, includeResolved: boolean = false): Promise<Prisma.issue[]> {
        const whereClause: Prisma.issueWhereInput = { teamId };
        
        if (!includeResolved) {
            whereClause.status = { 
                notIn: [IssueStatus.Resolved, IssueStatus.Rejected] 
            };
        }
        
        return await this.prisma.issue.findMany({
            where: whereClause,
            include: this.getDefaultInclude(),
            orderBy: { createdAt: 'desc' },
        });
    }
}

// Export factory creator function
export const createIssueDbFactory = (prisma: PrismaClient) => 
    IssueDbFactory.getInstance('issue', prisma);

// Export the class for type usage
export { IssueDbFactory as IssueDbFactoryClass };