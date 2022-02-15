import { CODE, reportCreate, ReportFor, reportUpdate } from "@local/shared";
import { CustomError } from "../error";
import { DeleteOneInput, Report, ReportCreateInput, ReportUpdateInput, Success } from "../schema/types";
import { PrismaType, RecursivePartial } from "types";
import { hasProfanity } from "../utils/censor";
import { FormatConverter, MODEL_TYPES } from "./base";
import { report } from "@prisma/client";

//==============================================================
/* #region Custom Components */
//==============================================================

const forMapper = {
    [ReportFor.Comment]: 'commentId',
    [ReportFor.Organization]: 'organizationId',
    [ReportFor.Project]: 'projectId',
    [ReportFor.Routine]: 'routineId',
    [ReportFor.Standard]: 'standardId',
    [ReportFor.Tag]: 'tagId',
    [ReportFor.User]: 'userId',
}

/**
 * Component for formatting between graphql and prisma types
 */
 export const reportFormatter = (): FormatConverter<Report, report> => {
    return {
        toDB: (obj: RecursivePartial<Report>): RecursivePartial<report> => {
            let modified = obj;
            // Remove calculated fields
            delete modified.isOwn;
            // Add userId for calculating isOwn
            return { ...obj, userId: true } as any;
        },
        toGraphQL: (obj: RecursivePartial<report>): RecursivePartial<Report> => {
            // Dont't show who submitted the report
            let modified = obj;
            delete modified.userId;
            return modified;
        },
    }
}

/**
 * Handles the authorized adding, updating, and deleting of reports.
 * Only users can add reports, and they can only do so once per object. 
 * They can technically report their own objects, but why would they?
 */
const reporter = (format: FormatConverter<Report, report>, prisma: PrismaType) => ({
    async create(
        userId: string,
        input: ReportCreateInput,
    ): Promise<RecursivePartial<Report>> {
        // Check for valid arguments
        reportCreate.validateSync(input, { abortEarly: false });
        // Check for censored words
        if (hasProfanity(input.reason, input.details)) throw new CustomError(CODE.BannedWord);
        // Add report
        const report = await prisma.report.create({
            data: {
                reason: input.reason,
                details: input.details,
                from: { connect: { id: userId } },
                [forMapper[input.createdFor]]: input.createdForId,
            }
        })
        // Return report with "isOwn" field
        return { ...format.toGraphQL(report), isOwn: true };
    },
    async update(
        userId: string,
        input: ReportUpdateInput,
    ): Promise<RecursivePartial<Report>> {
        // Check for valid arguments
        reportUpdate.validateSync(input, { abortEarly: false });
        // Check for censored words
        if (hasProfanity(input.reason, input.details)) throw new CustomError(CODE.BannedWord);
        // Find report
        let report = await prisma.report.findFirst({
            where: {
                id: input.id,
                userId,
            }
        })
        if (!report) throw new CustomError(CODE.ErrorUnknown);
        // Update report
        report = await prisma.report.update({
            where: { id: report.id },
            data: {
                reason: input.reason ?? undefined,
                details: input.details,
            }
        });
        // Format and add supplemental/calculated fields
        const formatted = await this.supplementalFields(userId, [format.toGraphQL(report)], {});
        return formatted[0];
    },
    async delete(userId: string, input: DeleteOneInput): Promise<Success> {
        const result = await prisma.report.deleteMany({
            where: {
                AND: [
                    { id: input.id },
                    { userId },
                ]
            }
        })
        return { success: Boolean(result.count) };
    },
    /**
     * Supplemental fields are isOwn
     */
    async supplementalFields(
        userId: string | null | undefined, // Of the user making the request
        objects: RecursivePartial<Report & { fromId: string | null }>[],
        known: { [x: string]: any[] }, // Known values (i.e. don't need to query), in same order as objects
    ): Promise<RecursivePartial<Report>[]> {
        // If userId not provided, return the input with isOwn false
        if (!userId) return objects.map(x => ({ ...x, isOwn: false }));
        // Check is isOwn is provided
        if (known.isOwn) objects = objects.map((x, i) => ({ ...x, isOwn: known.isOwn[i] }));
        // Otherwise, query for isOwn
        else objects = objects.map((x) => ({ ...x, isOwn: x.fromId === userId }));
        return objects;
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function ReportModel(prisma: PrismaType) {
    const model = MODEL_TYPES.Report;
    const format = reportFormatter();

    return {
        prisma,
        model,
        ...format,
        ...reporter(format, prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================