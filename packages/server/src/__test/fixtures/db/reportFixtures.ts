import { generatePK, generatePublicId } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for Report model - used for seeding test data
 */

export class ReportDbFactory {
    static createMinimal(
        createdById: string,
        reason: string,
        overrides?: Partial<Prisma.reportCreateInput>,
    ): Prisma.reportCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            createdBy: { connect: { id: createdById } },
            reason,
            language: "en",
            isDeleted: false,
            ...overrides,
        };
    }

    static createForObject(
        createdById: string,
        objectId: string,
        objectType: string,
        reason: string,
        overrides?: Partial<Prisma.reportCreateInput>,
    ): Prisma.reportCreateInput {
        const base = this.createMinimal(createdById, reason, overrides);
        
        // Add the appropriate connection based on object type
        const connections: Record<string, any> = {
            Api: { api: { connect: { id: objectId } } },
            ChatMessage: { chatMessage: { connect: { id: objectId } } },
            Code: { code: { connect: { id: objectId } } },
            Comment: { comment: { connect: { id: objectId } } },
            Issue: { issue: { connect: { id: objectId } } },
            Note: { note: { connect: { id: objectId } } },
            Post: { post: { connect: { id: objectId } } },
            Project: { project: { connect: { id: objectId } } },
            Prompt: { prompt: { connect: { id: objectId } } },
            Question: { question: { connect: { id: objectId } } },
            Quiz: { quiz: { connect: { id: objectId } } },
            Routine: { routine: { connect: { id: objectId } } },
            SmartContract: { smartContract: { connect: { id: objectId } } },
            Standard: { standard: { connect: { id: objectId } } },
            Tag: { tag: { connect: { id: objectId } } },
            Team: { team: { connect: { id: objectId } } },
            User: { user: { connect: { id: objectId } } },
        };

        return {
            ...base,
            ...(connections[objectType] || {}),
        };
    }

    static createWithDetails(
        createdById: string,
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
        createdById: string,
        reportId: string,
        overrides?: Partial<Prisma.ReportResponseCreateInput>,
    ): Prisma.ReportResponseCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            createdBy: { connect: { id: createdById } },
            report: { connect: { id: reportId } },
            actionSuggested: "NoAction",
            language: "en",
            ...overrides,
        };
    }

    static createWithDetails(
        createdById: string,
        reportId: string,
        actionSuggested: string,
        details: string,
        overrides?: Partial<Prisma.ReportResponseCreateInput>,
    ): Prisma.ReportResponseCreateInput {
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
        createdById: string;
        objects: Array<{ id: string; type: string; reason: string; details?: string }>;
        withResponses?: Array<{ responderId: string; action: string; details?: string }>;
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

                const response = await prisma.reportResponse.create({ data: responseData });
                responses.push(response);
            }
        }
    }

    return { reports, responses };
}
