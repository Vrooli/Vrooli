import { generatePK, ReportSuggestedAction } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for ReportResponse model - used for seeding test data
 * These fixtures represent responses to reports, where users can suggest actions
 * like deletion, suspension, or marking reports as false positives
 */

// Consistent IDs for testing
export const reportResponseDbIds = {
    response1: generatePK(),
    response2: generatePK(),
    response3: generatePK(),
    response4: generatePK(),
    response5: generatePK(),
};

/**
 * Minimal report response data for database creation
 */
export const minimalReportResponseDb: Prisma.ReportResponseCreateInput = {
    id: reportResponseDbIds.response1,
    actionSuggested: ReportSuggestedAction.NonIssue,
    language: "en",
    createdBy: { connect: { id: "user_123" } },
    report: { connect: { id: "report_123" } },
};

/**
 * Report response with details
 */
export const reportResponseWithDetailsDb: Prisma.ReportResponseCreateInput = {
    id: reportResponseDbIds.response2,
    actionSuggested: ReportSuggestedAction.Delete,
    details: "This content violates community guidelines and should be removed immediately.",
    language: "en",
    createdBy: { connect: { id: "user_456" } },
    report: { connect: { id: "report_456" } },
};

/**
 * Complete report response with all features
 */
export const completeReportResponseDb: Prisma.ReportResponseCreateInput = {
    id: reportResponseDbIds.response3,
    actionSuggested: ReportSuggestedAction.SuspendUser,
    details: "User has repeatedly posted inappropriate content. Previous warnings have been ignored. Recommend temporary suspension.",
    language: "en",
    createdBy: { connect: { id: "user_789" } },
    report: { connect: { id: "report_789" } },
};

/**
 * False report response
 */
export const falseReportResponseDb: Prisma.ReportResponseCreateInput = {
    id: reportResponseDbIds.response4,
    actionSuggested: ReportSuggestedAction.FalseReport,
    details: "This report appears to be made in bad faith. The content does not violate any guidelines.",
    language: "en",
    createdBy: { connect: { id: "user_mod1" } },
    report: { connect: { id: "report_false" } },
};

/**
 * Hide until fixed response
 */
export const hideUntilFixedResponseDb: Prisma.ReportResponseCreateInput = {
    id: reportResponseDbIds.response5,
    actionSuggested: ReportSuggestedAction.HideUntilFixed,
    details: "Content has minor issues that can be corrected. Hide temporarily until author fixes.",
    language: "en",
    createdBy: { connect: { id: "user_mod2" } },
    report: { connect: { id: "report_fixable" } },
};

/**
 * Factory for creating report response database fixtures with overrides
 */
export class ReportResponseDbFactory {
    static createMinimal(
        createdById: string,
        reportId: string,
        overrides?: Partial<Prisma.ReportResponseCreateInput>,
    ): Prisma.ReportResponseCreateInput {
        return {
            id: generatePK(),
            actionSuggested: ReportSuggestedAction.NonIssue,
            language: "en",
            createdBy: { connect: { id: createdById } },
            report: { connect: { id: reportId } },
            ...overrides,
        };
    }

    static createWithDetails(
        createdById: string,
        reportId: string,
        actionSuggested: ReportSuggestedAction,
        details: string,
        overrides?: Partial<Prisma.ReportResponseCreateInput>,
    ): Prisma.ReportResponseCreateInput {
        return {
            id: generatePK(),
            actionSuggested,
            details,
            language: "en",
            createdBy: { connect: { id: createdById } },
            report: { connect: { id: reportId } },
            ...overrides,
        };
    }

    static createComplete(
        createdById: string,
        reportId: string,
        overrides?: Partial<Prisma.ReportResponseCreateInput>,
    ): Prisma.ReportResponseCreateInput {
        return {
            id: generatePK(),
            actionSuggested: ReportSuggestedAction.Delete,
            details: "Comprehensive response with detailed reasoning for the suggested action.",
            language: "en",
            createdBy: { connect: { id: createdById } },
            report: { connect: { id: reportId } },
            ...overrides,
        };
    }

    /**
     * Create response for specific action types
     */
    static createDelete(
        createdById: string,
        reportId: string,
        details?: string,
        overrides?: Partial<Prisma.ReportResponseCreateInput>,
    ): Prisma.ReportResponseCreateInput {
        return this.createWithDetails(
            createdById,
            reportId,
            ReportSuggestedAction.Delete,
            details || "Content should be deleted due to policy violations.",
            overrides,
        );
    }

    static createSuspendUser(
        createdById: string,
        reportId: string,
        details?: string,
        overrides?: Partial<Prisma.ReportResponseCreateInput>,
    ): Prisma.ReportResponseCreateInput {
        return this.createWithDetails(
            createdById,
            reportId,
            ReportSuggestedAction.SuspendUser,
            details || "User behavior warrants temporary suspension.",
            overrides,
        );
    }

    static createFalseReport(
        createdById: string,
        reportId: string,
        details?: string,
        overrides?: Partial<Prisma.ReportResponseCreateInput>,
    ): Prisma.ReportResponseCreateInput {
        return this.createWithDetails(
            createdById,
            reportId,
            ReportSuggestedAction.FalseReport,
            details || "Report does not appear to be valid.",
            overrides,
        );
    }

    static createHideUntilFixed(
        createdById: string,
        reportId: string,
        details?: string,
        overrides?: Partial<Prisma.ReportResponseCreateInput>,
    ): Prisma.ReportResponseCreateInput {
        return this.createWithDetails(
            createdById,
            reportId,
            ReportSuggestedAction.HideUntilFixed,
            details || "Content needs corrections before being visible again.",
            overrides,
        );
    }

    static createNonIssue(
        createdById: string,
        reportId: string,
        details?: string,
        overrides?: Partial<Prisma.ReportResponseCreateInput>,
    ): Prisma.ReportResponseCreateInput {
        return this.createWithDetails(
            createdById,
            reportId,
            ReportSuggestedAction.NonIssue,
            details || "No action needed - content is within guidelines.",
            overrides,
        );
    }

    /**
     * Create response in different languages
     */
    static createMultiLanguage(
        createdById: string,
        reportId: string,
        language: string,
        overrides?: Partial<Prisma.ReportResponseCreateInput>,
    ): Prisma.ReportResponseCreateInput {
        const responses = {
            en: "Content violates community guidelines.",
            es: "El contenido viola las directrices de la comunidad.",
            fr: "Le contenu viole les directives de la communauté.",
            de: "Der Inhalt verstößt gegen die Community-Richtlinien.",
        };

        return this.createWithDetails(
            createdById,
            reportId,
            ReportSuggestedAction.Delete,
            responses[language as keyof typeof responses] || responses.en,
            { language, ...overrides },
        );
    }
}

/**
 * Helper to seed report responses for testing
 */
export async function seedReportResponses(
    prisma: any,
    options: {
        reportId: string;
        responses: Array<{
            responderId: string;
            action: ReportSuggestedAction;
            details?: string;
            language?: string;
        }>;
    },
) {
    const responses = [];

    for (const respData of options.responses) {
        const responseData = ReportResponseDbFactory.createWithDetails(
            respData.responderId,
            options.reportId,
            respData.action,
            respData.details || `Response suggesting ${respData.action}`,
            { language: respData.language || "en" },
        );

        const response = await prisma.report_response.create({
            data: responseData,
            include: {
                createdBy: {
                    select: {
                        id: true,
                        name: true,
                        handle: true,
                    },
                },
                report: {
                    select: {
                        id: true,
                        reason: true,
                        status: true,
                    },
                },
            },
        });
        responses.push(response);
    }

    return responses;
}

/**
 * Helper to create moderation scenarios for testing
 */
export async function seedModerationScenario(
    prisma: any,
    options: {
        reportId: string;
        moderators: Array<{ id: string; name: string }>;
        scenario: "consensus" | "disagreement" | "escalation";
    },
) {
    const responses = [];

    switch (options.scenario) {
        case "consensus":
            // All moderators agree to delete
            for (const mod of options.moderators) {
                const response = await prisma.report_response.create({
                    data: ReportResponseDbFactory.createDelete(
                        mod.id,
                        options.reportId,
                        `${mod.name} agrees - content should be removed.`,
                    ),
                });
                responses.push(response);
            }
            break;

        case "disagreement":
            // Moderators have different opinions
            const actions = [
                ReportSuggestedAction.Delete,
                ReportSuggestedAction.FalseReport,
                ReportSuggestedAction.HideUntilFixed,
            ];
            for (let i = 0; i < options.moderators.length; i++) {
                const mod = options.moderators[i];
                const action = actions[i % actions.length];
                const response = await prisma.report_response.create({
                    data: ReportResponseDbFactory.createWithDetails(
                        mod.id,
                        options.reportId,
                        action,
                        `${mod.name} suggests ${action}`,
                    ),
                });
                responses.push(response);
            }
            break;

        case "escalation":
            // Escalate to suspension
            for (const mod of options.moderators) {
                const response = await prisma.report_response.create({
                    data: ReportResponseDbFactory.createSuspendUser(
                        mod.id,
                        options.reportId,
                        `${mod.name} recommends user suspension due to repeat violations.`,
                    ),
                });
                responses.push(response);
            }
            break;
    }

    return responses;
}
