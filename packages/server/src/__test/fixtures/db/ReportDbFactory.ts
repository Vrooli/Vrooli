/* eslint-disable no-magic-numbers */
import { type Prisma, type PrismaClient, type report } from "@prisma/client";
import { generatePublicId, ReportStatus, ReportSuggestedAction } from "@vrooli/shared";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type {
    DbTestFixtures,
    RelationConfig,
    TestScenario,
} from "./types.js";

interface ReportRelationConfig extends RelationConfig {
    createdById?: bigint;
    targetType?: "resourceVersion" | "chatMessage" | "comment" | "issue" | "tag" | "team" | "user";
    targetId?: bigint;
    withResponses?: Array<{
        createdById: bigint;
        actionSuggested: ReportSuggestedAction;
        details?: string;
    }>;
}

/**
 * Enhanced database fixture factory for Report model
 * Provides comprehensive testing capabilities for content reporting
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for polymorphic relationships (various content types)
 * - Report status workflow testing
 * - Response management
 * - Language support
 * - Predefined test scenarios
 * - Report resolution testing
 */
export class ReportDbFactory extends EnhancedDatabaseFactory<
    report,
    Prisma.reportCreateInput,
    Prisma.reportInclude,
    Prisma.reportUpdateInput
> {
    protected scenarios: Record<string, TestScenario> = {};
    constructor(prisma: PrismaClient) {
        super("report", prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.report;
    }

    /**
     * Get complete test fixtures for Report model
     */
    protected getFixtures(): DbTestFixtures<Prisma.reportCreateInput, Prisma.reportUpdateInput> {
        return {
            minimal: {
                id: this.generateId(),
                publicId: generatePublicId(),
                reason: "Inappropriate content",
                language: "en",
                status: ReportStatus.Open,
                createdBy: { connect: { id: this.generateId() } },
                // Must have at least one target
                user: { connect: { id: this.generateId() } },
            },
            complete: {
                id: this.generateId(),
                publicId: generatePublicId(),
                reason: "Spam and misleading information",
                details: "This user is posting spam links and misleading information about cryptocurrency investments. Multiple posts contain phishing links.",
                language: "en",
                status: ReportStatus.Open,
                createdBy: { connect: { id: this.generateId() } },
                user: { connect: { id: this.generateId() } },
                createdAt: new Date(),
                updatedAt: new Date(),
            },
            invalid: {
                missingRequired: {
                    // Missing id, publicId, reason, language, status, and any target
                },
                invalidTypes: {
                    id: "not-a-snowflake",
                    publicId: 123, // Should be string
                    reason: null, // Should be string
                    language: true, // Should be string
                    status: "open", // Should be ReportStatus enum
                    createdById: null, // Should be string
                },
                noTarget: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    reason: "Test reason",
                    language: "en",
                    status: ReportStatus.Open,
                    createdBy: { connect: { id: this.generateId() } },
                    // No target specified
                },
                multipleTargets: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    reason: "Test reason",
                    language: "en",
                    status: ReportStatus.Open,
                    createdBy: { connect: { id: this.generateId() } },
                    user: { connect: { id: this.generateId() } },
                    team: { connect: { id: this.generateId() } }, // Multiple targets not allowed
                },
                reasonTooLong: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    reason: "a".repeat(129), // Exceeds 128 character limit
                    language: "en",
                    status: ReportStatus.Open,
                    createdBy: { connect: { id: this.generateId() } },
                    user: { connect: { id: this.generateId() } },
                },
                detailsTooLong: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    reason: "Valid reason",
                    details: "a".repeat(8193), // Exceeds 8192 character limit
                    language: "en",
                    status: ReportStatus.Open,
                    createdBy: { connect: { id: this.generateId() } },
                    user: { connect: { id: this.generateId() } },
                },
                invalidLanguageCode: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    reason: "Valid reason",
                    language: "invalid", // Should be valid ISO language code
                    status: ReportStatus.Open,
                    createdBy: { connect: { id: this.generateId() } },
                    user: { connect: { id: this.generateId() } },
                },
            },
            edgeCases: {
                reportWithoutDetails: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    reason: "Offensive language",
                    language: "en",
                    status: ReportStatus.Open,
                    createdBy: { connect: { id: this.generateId() } },
                    user: { connect: { id: this.generateId() } },
                },
                closedReport: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    reason: "Spam",
                    language: "en",
                    status: ReportStatus.ClosedDeleted,
                    createdBy: { connect: { id: this.generateId() } },
                    user: { connect: { id: this.generateId() } },
                },
                falseReport: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    reason: "Harassment",
                    details: "Actually just a disagreement",
                    language: "en",
                    status: ReportStatus.ClosedFalseReport,
                    createdBy: { connect: { id: this.generateId() } },
                    user: { connect: { id: this.generateId() } },
                },
                reportOfComment: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    reason: "Hate speech",
                    language: "en",
                    status: ReportStatus.Open,
                    createdBy: { connect: { id: this.generateId() } },
                    comment: { connect: { id: this.generateId() } },
                },
                reportOfChatMessage: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    reason: "Inappropriate content",
                    language: "en",
                    status: ReportStatus.Open,
                    createdBy: { connect: { id: this.generateId() } },
                    chatMessageId: this.generateId(),
                },
                reportOfTeam: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    reason: "Impersonation",
                    language: "en",
                    status: ReportStatus.Open,
                    createdBy: { connect: { id: this.generateId() } },
                    team: { connect: { id: this.generateId() } },
                },
                multiLanguageReport: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    reason: "Contenido inapropiado",
                    details: "Este usuario está publicando contenido ofensivo",
                    language: "es",
                    status: ReportStatus.Open,
                    createdBy: { connect: { id: this.generateId() } },
                    user: { connect: { id: this.generateId() } },
                },
            },
            updates: {
                minimal: {
                    status: ReportStatus.ClosedNonIssue,
                },
                complete: {
                    status: ReportStatus.ClosedSuspended,
                    details: "Updated details after investigation",
                    updatedAt: new Date(),
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.reportCreateInput>): Prisma.reportCreateInput {
        return {
            id: this.generateId(),
            publicId: generatePublicId(),
            reason: "Inappropriate content",
            language: "en",
            status: ReportStatus.Open,
            createdBy: { connect: { id: this.generateId() } },
            user: { connect: { id: this.generateId() } }, // Default to user report
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.reportCreateInput>): Prisma.reportCreateInput {
        return {
            id: this.generateId(),
            publicId: generatePublicId(),
            reason: "Multiple violations",
            details: "This content violates multiple community guidelines including spam, harassment, and misleading information.",
            language: "en",
            status: ReportStatus.Open,
            createdBy: { connect: { id: this.generateId() } },
            user: { connect: { id: this.generateId() } },
            createdAt: new Date(),
            updatedAt: new Date(),
            ...overrides,
        };
    }

    /**
     * Initialize test scenarios
     */
    protected initializeScenarios(): void {
        this.scenarios = {
            spamReport: {
                name: "spamReport",
                description: "Report for spam content",
                config: {
                    overrides: {
                        reason: "Spam",
                        details: "Posting repetitive promotional content",
                    },
                },
            },
            harassmentReport: {
                name: "harassmentReport",
                description: "Report for harassment",
                config: {
                    overrides: {
                        reason: "Harassment",
                        details: "Targeted harassment and bullying behavior",
                    },
                },
            },
            copyrightReport: {
                name: "copyrightReport",
                description: "Report for copyright violation",
                config: {
                    overrides: {
                        reason: "Copyright violation",
                        details: "Using copyrighted content without permission",
                    },
                    targetType: "resourceVersion",
                },
            },
            impersonationReport: {
                name: "impersonationReport",
                description: "Report for impersonation",
                config: {
                    overrides: {
                        reason: "Impersonation",
                        details: "Pretending to be another person or organization",
                    },
                    targetType: "team",
                },
            },
            resolvedReport: {
                name: "resolvedReport",
                description: "Report that has been resolved",
                config: {
                    overrides: {
                        reason: "Offensive content",
                        status: ReportStatus.ClosedDeleted,
                    },
                    withResponses: [
                        {
                            createdBy: { connect: { id: this.generateId() } },
                            actionSuggested: ReportSuggestedAction.Delete,
                            details: "Content violated terms of service and has been removed",
                        },
                    ],
                },
            },
            disputedReport: {
                name: "disputedReport",
                description: "Report with multiple conflicting responses",
                config: {
                    overrides: {
                        reason: "Misleading information",
                    },
                    withResponses: [
                        {
                            createdBy: { connect: { id: this.generateId() } },
                            actionSuggested: ReportSuggestedAction.Delete,
                            details: "This is clearly misleading",
                        },
                        {
                            createdBy: { connect: { id: this.generateId() } },
                            actionSuggested: ReportSuggestedAction.NonIssue,
                            details: "This is just an opinion, not misleading",
                        },
                    ],
                },
            },
        };
    }

    protected getDefaultInclude(): Prisma.reportInclude {
        return {
            createdBy: {
                select: {
                    id: true,
                    publicId: true,
                    name: true,
                    handle: true,
                },
            },
            resourceVersion: {
                select: {
                    id: true,
                    publicId: true,
                },
            },
            chatMessage: {
                select: {
                    id: true,
                    publicId: true,
                },
            },
            comment: {
                select: {
                    id: true,
                    publicId: true,
                },
            },
            issue: {
                select: {
                    id: true,
                    publicId: true,
                    title: true,
                },
            },
            tag: {
                select: {
                    id: true,
                    publicId: true,
                    tag: true,
                },
            },
            team: {
                select: {
                    id: true,
                    publicId: true,
                    name: true,
                    handle: true,
                },
            },
            user: {
                select: {
                    id: true,
                    publicId: true,
                    name: true,
                    handle: true,
                },
            },
            responses: {
                select: {
                    id: true,
                    actionSuggested: true,
                    details: true,
                    language: true,
                    createdAt: true,
                    createdBy: {
                        select: {
                            id: true,
                            publicId: true,
                            name: true,
                        },
                    },
                },
            },
            _count: {
                select: {
                    responses: true,
                    subscriptions: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.reportCreateInput,
        config: ReportRelationConfig,
        tx: any,
    ): Promise<Prisma.reportCreateInput> {
        const data = { ...baseData };

        // Handle creator association
        if (config.createdById) {
            data.createdById = config.createdById;
        }

        // Handle target type
        if (config.targetType && config.targetId) {
            // Clear any existing targets
            delete data.resourceVersionId;
            delete data.chatMessageId;
            delete data.commentId;
            delete data.issueId;
            delete data.tagId;
            delete data.teamId;
            delete data.userId;

            // Set the appropriate target
            switch (config.targetType) {
                case "resourceVersion":
                    data.resourceVersionId = config.targetId;
                    break;
                case "chatMessage":
                    data.chatMessageId = config.targetId;
                    break;
                case "comment":
                    data.commentId = config.targetId;
                    break;
                case "issue":
                    data.issueId = config.targetId;
                    break;
                case "tag":
                    data.tagId = config.targetId;
                    break;
                case "team":
                    data.teamId = config.targetId;
                    break;
                case "user":
                    data.userId = config.targetId;
                    break;
            }
        }

        // Handle responses
        if (config.withResponses && Array.isArray(config.withResponses)) {
            data.responses = {
                create: config.withResponses.map(response => ({
                    id: this.generateId(),
                    createdById: response.createdById,
                    actionSuggested: response.actionSuggested,
                    details: response.details,
                    language: data.language,
                })),
            };
        }

        return data;
    }

    /**
     * Create a report for specific content
     */
    async createReportFor(
        targetType: "resourceVersion" | "chatMessage" | "comment" | "issue" | "tag" | "team" | "user",
        targetId: string,
        createdById: string,
        reason: string,
        details?: string,
    ): Promise<Prisma.report> {
        return await this.createWithRelations({
            overrides: { reason, details },
            createdById,
            targetType,
            targetId,
        });
    }

    /**
     * Create common report types
     */
    async createSpamReport(targetType: string, targetId: string, createdById: string): Promise<Prisma.report> {
        return await this.createReportFor(
            targetType as any,
            targetId,
            createdById,
            "Spam",
            "Repetitive promotional content",
        );
    }

    async createHarassmentReport(targetType: string, targetId: string, createdById: string): Promise<Prisma.report> {
        return await this.createReportFor(
            targetType as any,
            targetId,
            createdById,
            "Harassment",
            "Targeted harassment or bullying",
        );
    }

    async createInappropriateContentReport(targetType: string, targetId: string, createdById: string): Promise<Prisma.report> {
        return await this.createReportFor(
            targetType as any,
            targetId,
            createdById,
            "Inappropriate content",
            "Content violates community guidelines",
        );
    }

    /**
     * Update report status
     */
    async closeReport(
        reportId: string,
        status: ReportStatus,
        responseDetails?: string,
    ): Promise<Prisma.report> {
        return await this.prisma.report.update({
            where: { id: reportId },
            data: {
                status,
                updatedAt: new Date(),
            },
            include: this.getDefaultInclude(),
        });
    }

    protected async checkModelConstraints(record: Prisma.report): Promise<string[]> {
        const violations: string[] = [];

        // Check that only one target is specified
        const targetCount = [
            record.resourceVersionId,
            record.chatMessageId,
            record.commentId,
            record.issueId,
            record.tagId,
            record.teamId,
            record.userId,
        ].filter(Boolean).length;

        if (targetCount === 0) {
            violations.push("Report must have exactly one target");
        } else if (targetCount > 1) {
            violations.push("Report cannot have multiple targets");
        }

        // Check reason length
        if (!record.reason || record.reason.length === 0) {
            violations.push("Report reason cannot be empty");
        }

        if (record.reason && record.reason.length > 128) {
            violations.push("Report reason exceeds maximum length of 128 characters");
        }

        // Check details length
        if (record.details && record.details.length > 8192) {
            violations.push("Report details exceed maximum length of 8192 characters");
        }

        // Check language code format
        if (!record.language || !/^[a-z]{2,3}$/.test(record.language)) {
            violations.push("Invalid language code format");
        }

        // Check publicId uniqueness
        const duplicate = await this.prisma.report.findFirst({
            where: {
                publicId: record.publicId,
                id: { not: record.id },
            },
        });
        if (duplicate) {
            violations.push("Public ID must be unique");
        }

        // Check creator exists
        if (record.createdById) {
            const creator = await this.prisma.user.findUnique({
                where: { id: record.createdById },
            });
            if (!creator) {
                violations.push("Report creator does not exist");
            }
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {
            responses: true,
            subscriptions: true,
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.report,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[],
    ): Promise<void> {
        // Helper to check if a relation should be deleted
        const shouldDelete = (relation: string) =>
            !includeOnly || includeOnly.includes(relation);

        // Delete responses
        if (shouldDelete("responses") && record.responses?.length) {
            await tx.report_response.deleteMany({
                where: { reportId: record.id },
            });
        }

        // Delete notification subscriptions
        if (shouldDelete("subscriptions") && record.subscriptions?.length) {
            await tx.notification_subscription.deleteMany({
                where: { reportId: record.id },
            });
        }
    }

    /**
     * Create a report with workflow progression
     */
    async createReportWithWorkflow(
        targetType: string,
        targetId: string,
        workflow: Array<{
            status: ReportStatus;
            response?: {
                createdById: string;
                actionSuggested: ReportSuggestedAction;
                details?: string;
            };
        }>,
    ): Promise<Prisma.report> {
        // Create initial report
        const report = await this.createReportFor(
            targetType as any,
            targetId,
            this.generateId(),
            "Test report for workflow",
            "Testing report workflow progression",
        );

        // Apply workflow steps
        for (const step of workflow) {
            // Add response if provided
            if (step.response) {
                await this.prisma.report_response.create({
                    data: {
                        id: this.generateId(),
                        reportId: report.id,
                        createdById: step.response.createdById,
                        actionSuggested: step.response.actionSuggested,
                        details: step.response.details,
                        language: report.language,
                    },
                });
            }

            // Update status
            await this.prisma.report.update({
                where: { id: report.id },
                data: { status: step.status },
            });
        }

        return await this.prisma.report.findUnique({
            where: { id: report.id },
            include: this.getDefaultInclude(),
        }) as Prisma.report;
    }

    /**
     * Get report statistics
     */
    async getReportStats(options?: {
        startDate?: Date;
        endDate?: Date;
        targetType?: string;
    }): Promise<{
        total: number;
        byStatus: Record<string, number>;
        byReason: Record<string, number>;
        byTargetType: Record<string, number>;
    }> {
        const where: any = {};

        if (options?.startDate || options?.endDate) {
            where.createdAt = {};
            if (options.startDate) {
                where.createdAt.gte = options.startDate;
            }
            if (options.endDate) {
                where.createdAt.lte = options.endDate;
            }
        }

        const reports = await this.prisma.report.findMany({
            where,
            select: {
                status: true,
                reason: true,
                resourceVersionId: true,
                chatMessageId: true,
                commentId: true,
                issueId: true,
                tagId: true,
                teamId: true,
                userId: true,
            },
        });

        const byStatus: Record<string, number> = {};
        const byReason: Record<string, number> = {};
        const byTargetType: Record<string, number> = {};

        reports.forEach(report => {
            // Count by status
            byStatus[report.status] = (byStatus[report.status] || 0) + 1;

            // Count by reason
            byReason[report.reason] = (byReason[report.reason] || 0) + 1;

            // Determine target type
            let targetType = "unknown";
            if (report.resourceVersionId) targetType = "resourceVersion";
            else if (report.chatMessageId) targetType = "chatMessage";
            else if (report.commentId) targetType = "comment";
            else if (report.issueId) targetType = "issue";
            else if (report.tagId) targetType = "tag";
            else if (report.teamId) targetType = "team";
            else if (report.userId) targetType = "user";

            byTargetType[targetType] = (byTargetType[targetType] || 0) + 1;
        });

        return {
            total: reports.length,
            byStatus,
            byReason,
            byTargetType,
        };
    }
}

// Export factory creator function
export const createReportDbFactory = (prisma: PrismaClient) =>
    new ReportDbFactory(prisma);

// Export the class for type usage
export { ReportDbFactory as ReportDbFactoryClass };
