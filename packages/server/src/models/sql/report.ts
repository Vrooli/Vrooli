import { CODE, omit, ReportFor, reportsCreate, reportsUpdate } from "@local/shared";
import { CustomError } from "../../error";
import { Count, Report, ReportCreateInput, ReportSearchInput, ReportSortBy, ReportUpdateInput } from "../../schema/types";
import { PrismaType, RecursivePartial } from "../../types";
import { validateProfanity } from "../../utils/censor";
import { CUDInput, CUDResult, FormatConverter, modelToGraphQL, PartialGraphQLInfo, Searcher, selectHelper, ValidateMutationsInput } from "./base";
import { genErrorCode } from "../../logger";

//==============================================================
/* #region Custom Components */
//==============================================================

const supplementalFields = ['isOwn'];
export const reportFormatter = (): FormatConverter<Report, any> => ({
    relationshipMap: { '__typename': 'Report' },
    removeJoinTables: (data) => {
        // Remove userId to hide who submitted the report
        let { userId, ...rest } = data;
        return rest;
    },
    removeSupplementalFields: (partial) => {
        const omitted = omit(partial, supplementalFields)
        // Add userId field so we can calculate isOwn
        return { ...omitted, userId: true }
    },
    async addSupplementalFields({ objects, partial, prisma, userId }): Promise<RecursivePartial<Report>[]> {
        // Query for isOwn
        if (partial.isOwn) objects = objects.map((x) => ({ ...x, isOwn: Boolean(userId) && x.fromId === userId }));
        // Convert Prisma objects to GraphQL objects
        return objects as RecursivePartial<Report>[];
    },
})

export const reportSearcher = (): Searcher<ReportSearchInput> => ({
    defaultSort: ReportSortBy.DateCreatedDesc,
    getSortQuery: (sortBy: string): any => {
        return {
            [ReportSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [ReportSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [ReportSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [ReportSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string): any => {
        const insensitive = ({ contains: searchString.trim(), mode: 'insensitive' });
        return ({
            OR: [
                { reason: { ...insensitive } },
                { details: { ...insensitive } },
            ]
        })
    },
    customQueries(input: ReportSearchInput): { [x: string]: any } {
        return {
            ...(input.languages !== undefined ? { translations: { some: { language: { in: input.languages } } } } : {}),
            ...(input.userId !== undefined ? { userId: input.userId } : {}),
            ...(input.organizationId !== undefined ? { organizationId: input.organizationId } : {}),
            ...(input.projectId !== undefined ? { projectId: input.projectId } : {}),
            ...(input.routineId !== undefined ? { routineId: input.routineId } : {}),
            ...(input.standardId !== undefined ? { standardId: input.standardId } : {}),
            ...(input.tagId !== undefined ? { tagId: input.tagId } : {}),
        }
    },
})

export const reportVerifier = () => ({
    // TODO not sure if report should have profanity check, since someone might 
    // just be trying to submit a report for a profane word
    profanityCheck(data: (ReportCreateInput | ReportUpdateInput)[]): void {
        validateProfanity(data.map((d: any) => ({
            reason: d.reason,
            details: d.details,
        })));
    },
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

export const reportMutater = (prisma: PrismaType) => ({
    async toDBShapeAdd(userId: string | null, data: ReportCreateInput): Promise<any> {
        return {
            id: data.id,
            reason: data.reason,
            details: data.details,
            from: { connect: { id: userId } },
            [forMapper[data.createdFor]]: { connect: { id: data.createdForId } },
        }
    },
    async toDBShapeUpdate(userId: string | null, data: ReportUpdateInput): Promise<any> {
        return {
            reason: data.reason ?? undefined,
            details: data.details,
        }
    },
    async validateMutations({
        userId, createMany, updateMany, deleteMany
    }: ValidateMutationsInput<ReportCreateInput, ReportUpdateInput>): Promise<void> {
        if (!createMany && !updateMany && !deleteMany) return;
        if (!userId)
            throw new CustomError(CODE.Unauthorized, 'User must be logged in to perform CRUD operations', { code: genErrorCode('0083') });
        if (createMany) {
            reportsCreate.validateSync(createMany, { abortEarly: false });
            reportVerifier().profanityCheck(createMany);
            // Check if report already exists by user on object
            for (const input of createMany) {
                const existingReport = await prisma.report.count({
                    where: {
                        fromId: userId as string,
                        [`${forMapper[input.createdFor]}Id`]: input.createdForId,
                    }
                })
                if (existingReport > 0) {
                    throw new CustomError(CODE.ReportExists, 'You have already submitted a report for this object.', { code: genErrorCode('0084') });
                }
            }
        }
        if (updateMany) {
            reportsUpdate.validateSync(updateMany.map(u => u.data), { abortEarly: false });
            reportVerifier().profanityCheck(updateMany.map(u => u.data));
        }
    },
    async cud({ partialInfo, userId, createMany, updateMany, deleteMany }: CUDInput<ReportCreateInput, ReportUpdateInput>): Promise<CUDResult<Report>> {
        await this.validateMutations({ userId, createMany, updateMany, deleteMany });
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            // Loop through each create input
            for (const input of createMany) {
                // Call createData helper function
                const data = await this.toDBShapeAdd(userId, input);
                // Create object
                const currCreated = await prisma.report.create({ data, ...selectHelper(partialInfo) });
                // Convert to GraphQL
                const converted = modelToGraphQL(currCreated, partialInfo);
                // Add to created array
                created = created ? [...created, converted] : [converted];
            }
        }
        if (updateMany) {
            // Loop through each update input
            for (const input of updateMany) {
                // Find in database
                let object = await prisma.report.findFirst({
                    where: { ...input.where, userId }
                })
                if (!object) throw new CustomError(CODE.ErrorUnknown, 'Report not found.', { code: genErrorCode('0085') });
                // Update object
                const currUpdated = await prisma.report.update({
                    where: input.where,
                    data: await this.toDBShapeUpdate(userId, input.data),
                    ...selectHelper(partialInfo)
                });
                // Convert to GraphQL
                const converted = modelToGraphQL(currUpdated, partialInfo);
                // Add to updated array
                updated = updated ? [...updated, converted] : [converted];
            }
        }
        if (deleteMany) {
            deleted = await prisma.report.deleteMany({
                where: {
                    AND: [
                        { id: { in: deleteMany } },
                        { userId },
                    ]
                }
            })
        }
        return {
            created: createMany ? created : undefined,
            updated: updateMany ? updated : undefined,
            deleted: deleteMany ? deleted : undefined,
        };
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export const ReportModel = ({
    prismaObject: (prisma: PrismaType) => prisma.report,
    format: reportFormatter(),
    mutate: reportMutater,
    search: reportSearcher(),
    verify: reportVerifier(),
})

//==============================================================
/* #endregion Model */
//==============================================================