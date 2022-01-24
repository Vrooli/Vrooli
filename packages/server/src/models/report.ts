import { CODE, ReportFor } from "@local/shared";
import { CustomError } from "../error";
import { DeleteOneInput, Report, ReportInput, Success } from "../schema/types";
import { PrismaType, RecursivePartial } from "types";
import { hasProfanity } from "../utils/censor";
import { findByIder, FormatConverter, MODEL_TYPES } from "./base";
import { CommentDB } from "./comment";
import { OrganizationDB } from "./organization";
import { ProjectDB } from "./project";
import { RoutineDB } from "./routine";
import { StandardDB } from "./standard";
import { TagDB } from "./tag";
import { UserDB } from "./user";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type ReportRelationshipList = 'from' | 'comment' | 'organization' |
    'project' | 'routine' | 'standard' | 'tag' | 'user';
// Type 2. QueryablePrimitives
export type ReportQueryablePrimitives = Omit<Report, ReportRelationshipList>;
// Type 3. AllPrimitives
export type ReportAllPrimitives = ReportQueryablePrimitives & {
    fromId: string;
    commentId: string;
    organizationId: string;
    projectId: string;
    routineId: string;
    standardId: string;
    tagId: string;
    userId: string;
}
// type 4. Database shape
export type ReportDB = ReportAllPrimitives & {
    from: UserDB,
    comment: CommentDB,
    organization: OrganizationDB,
    project: ProjectDB,
    routine: RoutineDB,
    standard: StandardDB,
    tag: TagDB,
    user: UserDB,
}

//======================================================================================================================
/* #endregion Type Definitions */
//======================================================================================================================

//==============================================================
/* #region Custom Components */
//==============================================================

/**
 * Component for formatting between graphql and prisma types
 */
const formatter = (): FormatConverter<any, any> => ({
    toDB: (obj: RecursivePartial<Report>): RecursivePartial<any> => obj as any,
    toGraphQL: (obj: RecursivePartial<any>): RecursivePartial<Report> => obj as any
})

const forMapper = {
    [ReportFor.Comment]: 'comment',
    [ReportFor.Organization]: 'organization',
    [ReportFor.Project]: 'project',
    [ReportFor.Routine]: 'routine',
    [ReportFor.Standard]: 'standard',
    [ReportFor.Tag]: 'tag',
    [ReportFor.User]: 'user',
}

/**
 * Handles the authorized adding, updating, and deleting of reports.
 * Only users can add reports, and they can only do so once per object. 
 * They can technically report their own objects, but why would they?
 */
 const reporter = (prisma: PrismaType) => ({
    async addReport(
        userId: string, 
        input: ReportInput,
    ): Promise<any> {
        // Check for valid arguments
        if (!input.reason || input.reason.length < 1) throw new CustomError(CODE.InternalError, 'Reason must be provided');
        // Check for censored words
        if (hasProfanity(input.reason, input.details)) throw new CustomError(CODE.BannedWord);
        // Add report
        return await prisma.report.create({
            data: {
                reason: input.reason,
                details: input.details,
                from: { connect: { id: userId } },
                [forMapper[input.createdFor]]: input.forId,
            }
        })
    },
    async updateReport(
        userId: string, 
        input: ReportInput,
    ): Promise<any> {
        // Check for valid arguments
        if (!input.id) throw new CustomError(CODE.InternalError, 'No report id provided');
        if (!input.reason || input.reason.length < 1) throw new CustomError(CODE.InternalError, 'Reason must be provided');
        // Check for censored words
        if (hasProfanity(input.reason, input.details)) throw new CustomError(CODE.BannedWord);
        // Find report
        const report = await prisma.report.findFirst({
            where: {
                id: input.id,
                userId,
                [forMapper[input.createdFor]]: input.forId,
            }
        })
        if (!report) throw new CustomError(CODE.ErrorUnknown);
        // Update report
        return await prisma.report.update({
            where: { id: report.id },
            data: {
                reason: input.reason,
                details: input.details,
            }
        });
    },
    async deleteReport(userId: string, input: DeleteOneInput): Promise<Success> {
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
    const format = formatter();

    return {
        prisma,
        model,
        ...format,
        ...reporter(prisma),
        ...findByIder<Report, ReportDB>(model, format.toDB, prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================