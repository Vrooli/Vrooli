import { generatePK, generatePublicId, nanoid, ReportSuggestedAction } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
    TestScenario,
} from "./types.js";

interface ReportResponseRelationConfig extends RelationConfig {
    reportId?: string;
    createdById?: string;
}

/**
 * Enhanced database fixture factory for ReportResponse model
 * Provides comprehensive testing capabilities for report responses
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Report association management
 * - Suggested action testing
 * - Response details and language support
 * - Predefined test scenarios
 * - Unique constraint handling (one response per user per report)
 */
export class ReportResponseDbFactory extends EnhancedDatabaseFactory<
    Prisma.report_responseCreateInput,
    Prisma.report_responseCreateInput,
    Prisma.report_responseInclude,
    Prisma.report_responseUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super("report_response", prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.report_response;
    }

    /**
     * Get complete test fixtures for ReportResponse model
     */
    protected getFixtures(): DbTestFixtures<Prisma.report_responseCreateInput, Prisma.report_responseUpdateInput> {
        return {
            minimal: {
                id: generatePK().toString(),
                reportId: generatePK().toString(),
                createdById: generatePK().toString(),
                actionSuggested: ReportSuggestedAction.NonIssue,
            },
            complete: {
                id: generatePK().toString(),
                reportId: generatePK().toString(),
                createdById: generatePK().toString(),
                actionSuggested: ReportSuggestedAction.Delete,
                details: "After reviewing the reported content, I confirm it violates our community guidelines on hate speech. The content should be removed immediately and the user warned.",
                language: "en",
                createdAt: new Date(),
                updatedAt: new Date(),
            },
            invalid: {
                missingRequired: {
                    // Missing id, reportId, createdById, actionSuggested
                },
                invalidTypes: {
                    id: "not-a-snowflake",
                    reportId: null, // Should be string
                    createdById: 123, // Should be string
                    actionSuggested: "delete", // Should be ReportSuggestedAction enum
                    details: true, // Should be string
                    language: 456, // Should be string
                },
                detailsTooLong: {
                    id: generatePK().toString(),
                    reportId: generatePK().toString(),
                    createdById: generatePK().toString(),
                    actionSuggested: ReportSuggestedAction.Delete,
                    details: "a".repeat(8193), // Exceeds 8192 character limit
                },
                invalidLanguageCode: {
                    id: generatePK().toString(),
                    reportId: generatePK().toString(),
                    createdById: generatePK().toString(),
                    actionSuggested: ReportSuggestedAction.NonIssue,
                    language: "invalid", // Should be valid ISO language code
                },
                duplicateResponse: {
                    id: generatePK().toString(),
                    reportId: "existing-report-id",
                    createdById: "existing-user-id", // Same user responding to same report
                    actionSuggested: ReportSuggestedAction.FalseReport,
                },
            },
            edgeCases: {
                deleteResponse: {
                    id: generatePK().toString(),
                    reportId: generatePK().toString(),
                    createdById: generatePK().toString(),
                    actionSuggested: ReportSuggestedAction.Delete,
                    details: "Content violates terms of service",
                },
                falseReportResponse: {
                    id: generatePK().toString(),
                    reportId: generatePK().toString(),
                    createdById: generatePK().toString(),
                    actionSuggested: ReportSuggestedAction.FalseReport,
                    details: "This report is baseless and appears to be harassment",
                },
                hideUntilFixedResponse: {
                    id: generatePK().toString(),
                    reportId: generatePK().toString(),
                    createdById: generatePK().toString(),
                    actionSuggested: ReportSuggestedAction.HideUntilFixed,
                    details: "Content needs minor edits to comply with guidelines",
                },
                nonIssueResponse: {
                    id: generatePK().toString(),
                    reportId: generatePK().toString(),
                    createdById: generatePK().toString(),
                    actionSuggested: ReportSuggestedAction.NonIssue,
                    details: "Content does not violate any guidelines",
                },
                suspendUserResponse: {
                    id: generatePK().toString(),
                    reportId: generatePK().toString(),
                    createdById: generatePK().toString(),
                    actionSuggested: ReportSuggestedAction.SuspendUser,
                    details: "Repeated violations warrant account suspension",
                },
                minimalDetailsResponse: {
                    id: generatePK().toString(),
                    reportId: generatePK().toString(),
                    createdById: generatePK().toString(),
                    actionSuggested: ReportSuggestedAction.NonIssue,
                    // No details provided
                },
                multiLanguageResponse: {
                    id: generatePK().toString(),
                    reportId: generatePK().toString(),
                    createdById: generatePK().toString(),
                    actionSuggested: ReportSuggestedAction.Delete,
                    details: "Este contenido viola las directrices de la comunidad",
                    language: "es",
                },
            },
            updates: {
                minimal: {
                    actionSuggested: ReportSuggestedAction.Delete,
                },
                complete: {
                    actionSuggested: ReportSuggestedAction.SuspendUser,
                    details: "Updated after further investigation - user has multiple violations",
                    updatedAt: new Date(),
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.report_responseCreateInput>): Prisma.report_responseCreateInput {
        return {
            id: generatePK().toString(),
            reportId: generatePK().toString(),
            createdById: generatePK().toString(),
            actionSuggested: ReportSuggestedAction.NonIssue,
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.report_responseCreateInput>): Prisma.report_responseCreateInput {
        return {
            id: generatePK().toString(),
            reportId: generatePK().toString(),
            createdById: generatePK().toString(),
            actionSuggested: ReportSuggestedAction.Delete,
            details: "After thorough review, this content clearly violates our community guidelines and should be removed.",
            language: "en",
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
            adminResponse: {
                name: "adminResponse",
                description: "Admin responding to report with delete action",
                config: {
                    overrides: {
                        actionSuggested: ReportSuggestedAction.Delete,
                        details: "Admin review: Content violates ToS section 3.2",
                    },
                },
            },
            moderatorResponse: {
                name: "moderatorResponse",
                description: "Moderator suggesting content be hidden",
                config: {
                    overrides: {
                        actionSuggested: ReportSuggestedAction.HideUntilFixed,
                        details: "Content needs editing to remove personal information",
                    },
                },
            },
            communityResponse: {
                name: "communityResponse",
                description: "Community member indicating false report",
                config: {
                    overrides: {
                        actionSuggested: ReportSuggestedAction.FalseReport,
                        details: "I know the context - this is clearly not a violation",
                    },
                },
            },
            escalationResponse: {
                name: "escalationResponse",
                description: "Response escalating to user suspension",
                config: {
                    overrides: {
                        actionSuggested: ReportSuggestedAction.SuspendUser,
                        details: "User has 5+ confirmed violations in past month",
                    },
                },
            },
            dismissalResponse: {
                name: "dismissalResponse",
                description: "Response dismissing the report",
                config: {
                    overrides: {
                        actionSuggested: ReportSuggestedAction.NonIssue,
                        details: "This is a legitimate debate, not harassment",
                    },
                },
            },
            consensusResponse: {
                name: "consensusResponse",
                description: "Response aligning with previous responses",
                config: {
                    overrides: {
                        actionSuggested: ReportSuggestedAction.Delete,
                        details: "Agree with previous reviewers - clear violation",
                    },
                },
            },
        };
    }

    protected getDefaultInclude(): Prisma.report_responseInclude {
        return {
            report: {
                select: {
                    id: true,
                    publicId: true,
                    reason: true,
                    status: true,
                },
            },
            createdBy: {
                select: {
                    id: true,
                    publicId: true,
                    name: true,
                    handle: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.report_responseCreateInput,
        config: ReportResponseRelationConfig,
        tx: any,
    ): Promise<Prisma.report_responseCreateInput> {
        const data = { ...baseData };

        // Handle report association
        if (config.reportId) {
            data.reportId = config.reportId;
        }

        // Handle creator association
        if (config.createdById) {
            data.createdById = config.createdById;
        }

        return data;
    }

    /**
     * Create a response to a report
     */
    async createResponseToReport(
        reportId: string,
        createdById: string,
        actionSuggested: ReportSuggestedAction,
        details?: string,
    ): Promise<Prisma.report_response> {
        return await this.createWithRelations({
            overrides: {
                actionSuggested,
                details,
            },
            reportId,
            createdById,
        });
    }

    /**
     * Create multiple responses to simulate consensus/dispute
     */
    async createMultipleResponses(
        reportId: string,
        responses: Array<{
            createdById: string;
            actionSuggested: ReportSuggestedAction;
            details?: string;
        }>,
    ): Promise<Prisma.report_response[]> {
        const createdResponses: Prisma.report_response[] = [];

        for (const response of responses) {
            const created = await this.createResponseToReport(
                reportId,
                response.createdById,
                response.actionSuggested,
                response.details,
            );
            createdResponses.push(created);
        }

        return createdResponses;
    }

    /**
     * Create a consensus scenario (multiple users agree)
     */
    async createConsensusResponses(
        reportId: string,
        userIds: string[],
        actionSuggested: ReportSuggestedAction,
    ): Promise<Prisma.report_response[]> {
        const responses = userIds.map((userId, index) => ({
            createdById: userId,
            actionSuggested,
            details: `User ${index + 1} agrees this action is appropriate`,
        }));

        return await this.createMultipleResponses(reportId, responses);
    }

    /**
     * Create a dispute scenario (users disagree)
     */
    async createDisputeResponses(reportId: string, userIds: string[]): Promise<Prisma.report_response[]> {
        const actions = [
            ReportSuggestedAction.Delete,
            ReportSuggestedAction.NonIssue,
            ReportSuggestedAction.FalseReport,
            ReportSuggestedAction.HideUntilFixed,
        ];

        const responses = userIds.map((userId, index) => ({
            createdById: userId,
            actionSuggested: actions[index % actions.length],
            details: "Different perspective on the reported content",
        }));

        return await this.createMultipleResponses(reportId, responses);
    }

    protected async checkModelConstraints(record: Prisma.report_response): Promise<string[]> {
        const violations: string[] = [];
        
        // Check details length
        if (record.details && record.details.length > 8192) {
            violations.push("Response details exceed maximum length of 8192 characters");
        }

        // Check language code format if provided
        if (record.language && !/^[a-z]{2,3}$/.test(record.language)) {
            violations.push("Invalid language code format");
        }

        // Check report exists
        const report = await this.prisma.report.findUnique({
            where: { id: record.reportId },
        });
        if (!report) {
            violations.push("Report does not exist");
        }

        // Check creator exists
        const creator = await this.prisma.user.findUnique({
            where: { id: record.createdById },
        });
        if (!creator) {
            violations.push("Response creator does not exist");
        }

        // Check unique constraint (one response per user per report)
        const existingResponse = await this.prisma.report_response.findFirst({
            where: {
                reportId: record.reportId,
                createdById: record.createdById,
                id: { not: record.id },
            },
        });
        if (existingResponse) {
            violations.push("User has already responded to this report");
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {
            // Report responses don't have dependent records to cascade delete
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.report_response,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[],
    ): Promise<void> {
        // Report responses don't have dependent records
    }

    /**
     * Get response statistics for a report
     */
    async getResponseStats(reportId: string): Promise<{
        totalResponses: number;
        byAction: Record<string, number>;
        consensus: {
            hasConsensus: boolean;
            primaryAction?: ReportSuggestedAction;
            agreementPercentage?: number;
        };
    }> {
        const responses = await this.prisma.report_response.findMany({
            where: { reportId },
            select: { actionSuggested: true },
        });

        const byAction: Record<string, number> = {};
        responses.forEach(response => {
            byAction[response.actionSuggested] = (byAction[response.actionSuggested] || 0) + 1;
        });

        // Determine consensus
        const totalResponses = responses.length;
        let hasConsensus = false;
        let primaryAction: ReportSuggestedAction | undefined;
        let agreementPercentage: number | undefined;

        if (totalResponses > 0) {
            const maxCount = Math.max(...Object.values(byAction));
            const primaryActions = Object.entries(byAction)
                .filter(([_, count]) => count === maxCount)
                .map(([action]) => action as ReportSuggestedAction);

            if (primaryActions.length === 1) {
                primaryAction = primaryActions[0];
                agreementPercentage = (maxCount / totalResponses) * 100;
                hasConsensus = agreementPercentage >= 60; // 60% threshold for consensus
            }
        }

        return {
            totalResponses,
            byAction,
            consensus: {
                hasConsensus,
                primaryAction,
                agreementPercentage,
            },
        };
    }

    /**
     * Create a moderation workflow
     */
    async createModerationWorkflow(
        reportId: string,
        workflow: Array<{
            userId: string;
            role: "admin" | "moderator" | "community";
            waitHours?: number;
        }>,
    ): Promise<Prisma.report_response[]> {
        const responses: Prisma.report_response[] = [];

        for (const step of workflow) {
            // Determine action based on role
            let actionSuggested: ReportSuggestedAction;
            let details: string;

            switch (step.role) {
                case "admin":
                    actionSuggested = ReportSuggestedAction.Delete;
                    details = "Admin review complete - content violates policies";
                    break;
                case "moderator":
                    actionSuggested = ReportSuggestedAction.HideUntilFixed;
                    details = "Moderator review - needs content adjustment";
                    break;
                case "community":
                    actionSuggested = ReportSuggestedAction.NonIssue;
                    details = "Community member review - no violation found";
                    break;
            }

            const response = await this.createResponseToReport(
                reportId,
                step.userId,
                actionSuggested,
                details,
            );

            // Simulate time delay if specified
            if (step.waitHours) {
                const createdAt = new Date(Date.now() - step.waitHours * 60 * 60 * 1000);
                await this.prisma.report_response.update({
                    where: { id: response.id },
                    data: { createdAt },
                });
            }

            responses.push(response);
        }

        return responses;
    }
}

// Export factory creator function
export const createReportResponseDbFactory = (prisma: PrismaClient) => 
    ReportResponseDbFactory.getInstance("report_response", prisma);

// Export the class for type usage
export { ReportResponseDbFactory as ReportResponseDbFactoryClass };
