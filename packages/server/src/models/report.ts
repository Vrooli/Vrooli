import { Report, ReportInput } from "schema/types";
import { PrismaType, RecursivePartial } from "types";
import { creater, deleter, findByIder, FormatConverter, MODEL_TYPES, updater } from "./base";
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

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function ReportModel(prisma?: PrismaType) {
    const model = MODEL_TYPES.Report;
    const format = formatter();

    return {
        prisma,
        model,
        ...format,
        ...creater<ReportInput, Report, ReportDB>(model, format.toDB, prisma),
        ...deleter(model, prisma),
        ...findByIder<Report, ReportDB>(model, format.toDB, prisma),
        ...updater<ReportInput, Report, ReportDB>(model, format.toDB, prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================