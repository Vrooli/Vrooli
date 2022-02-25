import { CODE, reportCreate, ReportFor, reportUpdate } from "@local/shared";
import { CustomError } from "../error";
import { DeleteOneInput, Report, ReportCreateInput, ReportUpdateInput, Success } from "../schema/types";
import { PartialSelectConvert, PrismaType, RecursivePartial } from "types";
import { hasProfanity } from "../utils/censor";
import { FormatConverter, infoToPartialSelect, InfoType, MODEL_TYPES } from "./base";
import { report } from "@prisma/client";
import _ from "lodash";

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

type ReportFormatterType = FormatConverter<Report, report>;
/**
 * Component for formatting between graphql and prisma types
 */
export const reportFormatter = (): ReportFormatterType => {
    return {
        dbShape: (partial: PartialSelectConvert<Report>): PartialSelectConvert<report> => {
            let modified = partial;
            // Add userId for calculating isOwn
            return { ...modified, userId: partial.isOwn ? true : undefined };
        },
        dbPrune: (info: InfoType): PartialSelectConvert<report> => {
            // Convert GraphQL info object to a partial select object
            let modified = infoToPartialSelect(info);
            // Remove calculated fields
            let { isOwn, ...rest } = modified;
            modified = rest;
            return modified;
        },
        selectToDB: (info: InfoType): PartialSelectConvert<report> => {
            return reportFormatter().dbShape(reportFormatter().dbPrune(info));
        },
        selectToGraphQL: (obj: RecursivePartial<report>): RecursivePartial<Report> => {
            // Remove userId to hide who submitted the report
            let { userId, ...rest } = obj;
            // NOTE: "Add calculated fields" is done in the supplementalFields function. Allows results to batch their database queries.
            return rest;
        },
    }
}

/**
 * Handles the authorized adding, updating, and deleting of reports.
 * Only users can add reports, and they can only do so once per object. 
 * They can technically report their own objects, but why would they?
 */
const reporter = (format: ReportFormatterType, prisma: PrismaType) => ({
    async create(
        userId: string,
        input: ReportCreateInput,
    ): Promise<RecursivePartial<Report>> {
        // Check for valid arguments
        reportCreate.validateSync(input, { abortEarly: false });
        // Check for censored words
        this.profanityCheck(input);
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
        return { ...format.selectToGraphQL(report), isOwn: true };
    },
    async update(
        userId: string,
        input: ReportUpdateInput,
        info: InfoType,
    ): Promise<RecursivePartial<Report>> {
        // Check for valid arguments
        reportUpdate.validateSync(input, { abortEarly: false });
        // Check for censored words
        this.profanityCheck(input);
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
        const formatted = await this.supplementalFields(userId, [report], info);
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
        objects: RecursivePartial<any>[],
        info: InfoType, // GraphQL select info
    ): Promise<RecursivePartial<Report>[]> {
        // Convert GraphQL info object to a partial select object
        const partial = infoToPartialSelect(info);
        // Query for isOwn
        if (partial.isOwn) objects = objects.map((x) => ({ ...x, isOwn: Boolean(userId) && x.fromId === userId }));
        // Convert Prisma objects to GraphQL objects
        return objects.map(format.selectToGraphQL);
    },
    profanityCheck(data: ReportCreateInput | ReportUpdateInput): void {
        if (hasProfanity(data.reason, data.details)) throw new CustomError(CODE.BannedWord);
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