import { reportsCreate, reportsUpdate } from "@shared/validation";
import { ReportFor, ReportSortBy } from '@shared/consts';
import { CODE } from "@shared/consts";
import { omit } from '@shared/utils';
import { addSupplementalFieldsHelper, combineQueries, getSearchStringQueryHelper, modelToGraphQL, selectHelper } from "./builder";
import { CustomError, genErrorCode } from "../events";
import { Report, ReportSearchInput, ReportCreateInput, ReportUpdateInput, Count } from "../schema/types";
import { RecursivePartial, PrismaType } from "../types";
import { validateProfanity } from "../utils/censor";
import { FormatConverter, Searcher, ValidateMutationsInput, CUDInput, CUDResult, GraphQLModelType } from "./types";
import { Prisma, ReportStatus } from "@prisma/client";

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
        return addSupplementalFieldsHelper({
            objects,
            partial,
            resolvers: [
                ['isOwn', async () => objects.map((x) => Boolean(userId) && x.fromId === userId)],
            ]
        });
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
        return getSearchStringQueryHelper({
            searchString,
            resolver: ({ insensitive }) => ({
                OR: [
                    { reason: { ...insensitive } },
                    { details: { ...insensitive } },
                ]
            })
        })
    },
    customQueries(input: ReportSearchInput): { [x: string]: any } {
        return combineQueries([
            (input.languages !== undefined ? { translations: { some: { language: { in: input.languages } } } } : {}),
            (input.userId !== undefined ? { userId: input.userId } : {}),
            (input.organizationId !== undefined ? { organizationId: input.organizationId } : {}),
            (input.projectId !== undefined ? { projectId: input.projectId } : {}),
            (input.routineId !== undefined ? { routineId: input.routineId } : {}),
            (input.standardId !== undefined ? { standardId: input.standardId } : {}),
            (input.tagId !== undefined ? { tagId: input.tagId } : {}),
        ])
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
    async toDBShapeAdd(userId: string, data: ReportCreateInput): Promise<Prisma.reportUpsertArgs['create']> {
        return {
            id: data.id,
            language: data.language,
            reason: data.reason,
            details: data.details,
            status: ReportStatus.Open,
            from: { connect: { id: userId } },
            [forMapper[data.createdFor]]: { connect: { id: data.createdForId } },
        }
    },
    async toDBShapeUpdate(userId: string, data: ReportUpdateInput): Promise<Prisma.reportUpsertArgs['update']> {
        return {
            reason: data.reason ?? undefined,
            details: data.details,
        }
    },
    async validateMutations({
        userId, createMany, updateMany, deleteMany
    }: ValidateMutationsInput<ReportCreateInput, ReportUpdateInput>): Promise<void> {
        if (!createMany && !updateMany && !deleteMany) return;
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
    type: 'Report' as GraphQLModelType,
    verify: reportVerifier(),
})

//==============================================================
/* #endregion Model */
//==============================================================