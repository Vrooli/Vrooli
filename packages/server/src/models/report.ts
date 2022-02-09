import { CODE, reportCreate, ReportFor, reportUpdate } from "@local/shared";
import { CustomError } from "../error";
import { DeleteOneInput, Report, ReportCreateInput, ReportUpdateInput, Success } from "../schema/types";
import { PrismaType, RecursivePartial } from "types";
import { hasProfanity } from "../utils/censor";
import { MODEL_TYPES } from "./base";

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
 * Handles the authorized adding, updating, and deleting of reports.
 * Only users can add reports, and they can only do so once per object. 
 * They can technically report their own objects, but why would they?
 */
 const reporter = (prisma: PrismaType) => ({
    async create(
        userId: string, 
        input: ReportCreateInput,
    ): Promise<RecursivePartial<Report>> {
        // Check for valid arguments
        reportCreate.validateSync(input, { abortEarly: false });
        // Check for censored words
        if (hasProfanity(input.reason, input.details)) throw new CustomError(CODE.BannedWord);
        // Add report
        return await prisma.report.create({
            data: {
                reason: input.reason,
                details: input.details,
                from: { connect: { id: userId } },
                [forMapper[input.createdFor]]: input.createdForId,
            }
        })
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
        const report = await prisma.report.findFirst({
            where: {
                id: input.id,
                userId,
            }
        })
        if (!report) throw new CustomError(CODE.ErrorUnknown);
        // Update report
        return await prisma.report.update({
            where: { id: report.id },
            data: {
                reason: input.reason ?? undefined,
                details: input.details,
            }
        });
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
    }
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function ReportModel(prisma: PrismaType) {
    const model = MODEL_TYPES.Report;

    return {
        prisma,
        model,
        ...reporter(prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================