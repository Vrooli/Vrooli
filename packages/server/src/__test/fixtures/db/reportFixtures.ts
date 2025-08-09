import { generatePK, generatePublicId, ReportStatus, ReportSuggestedAction } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for Report model - used for seeding test data
 */

export class ReportDbFactory {
    static createMinimal(
        createdById: string | bigint,
        reason: string,
        overrides?: Partial<Prisma.reportCreateInput>,
    ): Prisma.reportCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            createdBy: { connect: { id: typeof createdById === "string" ? BigInt(createdById) : createdById } },
            reason,
            language: "en",
            status: ReportStatus.Open,
            ...overrides,
        };
    }

    static createForObject(
        createdById: string | bigint,
        objectId: string | bigint,
        objectType: string,
        reason: string,
        overrides?: Partial<Prisma.reportCreateInput>,
    ): Prisma.reportCreateInput {
        const base = this.createMinimal(createdById, reason, overrides);
        const id = typeof objectId === "string" ? BigInt(objectId) : objectId;
        
        // Add the appropriate connection based on object type
        const connections: Record<string, any> = {
            ResourceVersion: { resourceVersion: { connect: { id } } },
            ChatMessage: { chatMessage: { connect: { id } } },
            Comment: { comment: { connect: { id } } },
            Issue: { issue: { connect: { id } } },
            Tag: { tag: { connect: { id } } },
            Team: { team: { connect: { id } } },
            User: { user: { connect: { id } } },
        };

        return {
            ...base,
            ...(connections[objectType] || {}),
        };
    }

    static createWithDetails(
        createdById: string | bigint,
        reason: string,
        details: string,
        overrides?: Partial<Prisma.reportCreateInput>,
    ): Prisma.reportCreateInput {
        return this.createMinimal(createdById, reason, {
            details,
            ...overrides,
        });
    }
}

/**
 * Database fixtures for ReportResponse model
 */
export class ReportResponseDbFactory {
    static createMinimal(
        createdById: string | bigint,
        reportId: string | bigint,
        overrides?: Partial<Prisma.report_responseCreateInput>,
    ): Prisma.report_responseCreateInput {
        return {
            id: generatePK(),
            createdBy: { connect: { id: typeof createdById === "string" ? BigInt(createdById) : createdById } },
            report: { connect: { id: typeof reportId === "string" ? BigInt(reportId) : reportId } },
            actionSuggested: ReportSuggestedAction.NonIssue,
            language: "en",
            ...overrides,
        };
    }

    static createWithDetails(
        createdById: string | bigint,
        reportId: string | bigint,
        actionSuggested: ReportSuggestedAction,
        details: string,
        overrides?: Partial<Prisma.report_responseCreateInput>,
    ): Prisma.report_responseCreateInput {
        return this.createMinimal(createdById, reportId, {
            actionSuggested,
            details,
            ...overrides,
        });
    }
}

/**
 * Helper to seed reports for testing
 */
export async function seedReports(
    prisma: any,
    options: {
        createdById: string | bigint;
        objects: Array<{ id: string | bigint; type: string; reason: string; details?: string }>;
        withResponses?: Array<{ responderId: string | bigint; action: ReportSuggestedAction; details?: string }>;
    },
) {
    const reports = [];
    const responses = [];

    for (const obj of options.objects) {
        const reportData = obj.details
            ? ReportDbFactory.createForObject(
                options.createdById,
                obj.id,
                obj.type,
                obj.reason,
                { details: obj.details },
            )
            : ReportDbFactory.createForObject(
                options.createdById,
                obj.id,
                obj.type,
                obj.reason,
            );

        const report = await prisma.report.create({ data: reportData });
        reports.push(report);

        // Create responses if requested
        if (options.withResponses) {
            for (const respData of options.withResponses) {
                const responseData = respData.details
                    ? ReportResponseDbFactory.createWithDetails(
                        respData.responderId,
                        report.id,
                        respData.action,
                        respData.details,
                    )
                    : ReportResponseDbFactory.createMinimal(
                        respData.responderId,
                        report.id,
                        { actionSuggested: respData.action },
                    );

                const response = await prisma.report_response.create({ data: responseData });
                responses.push(response);
            }
        }
    }

    return { reports, responses };
}
