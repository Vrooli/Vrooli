import { reportsCreate, reportsUpdate } from "@shared/validation";
import { ReportFor, ReportSortBy } from '@shared/consts';
import { omit } from '@shared/utils';
import { addSupplementalFieldsHelper, combineQueries, getSearchStringQueryHelper } from "./builder";
import { Report, ReportSearchInput, ReportCreateInput, ReportUpdateInput } from "../schema/types";
import { RecursivePartial, PrismaType } from "../types";
import { FormatConverter, Searcher, CUDInput, CUDResult, GraphQLModelType } from "./types";
import { Prisma, ReportStatus } from "@prisma/client";
import { cudHelper } from "./actions";

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

export const reportValidator = () => ({
    // // TODO not sure if report should have profanity check, since someone might 
    // // just be trying to submit a report for a profane word
    // profanityCheck(data: (ReportCreateInput | ReportUpdateInput)[]): void {
    //     validateProfanity(data.map((d: any) => ({
    //         reason: d.reason,
    //         details: d.details,
    //     })));
    // },

    // Make sure user has only one report on object
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
    async shapeCreate(userId: string, data: ReportCreateInput): Promise<Prisma.reportUpsertArgs['create']> {
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
    async shapeUpdate(userId: string, data: ReportUpdateInput): Promise<Prisma.reportUpsertArgs['update']> {
        return {
            reason: data.reason ?? undefined,
            details: data.details,
        }
    },
    async cud(params: CUDInput<ReportCreateInput, ReportUpdateInput>): Promise<CUDResult<Report>> {
        return cudHelper({
            ...params,
            objectType: 'Report',
            prisma,
            prismaObject: (p) => p.report,
            yup: { yupCreate: reportsCreate, yupUpdate: reportsUpdate },
            shape: { shapeCreate: this.shapeCreate, shapeUpdate: this.shapeUpdate }
        })
    },
})

export const ReportModel = ({
    prismaObject: (prisma: PrismaType) => prisma.report,
    format: reportFormatter(),
    mutate: reportMutater,
    search: reportSearcher(),
    type: 'Report' as GraphQLModelType,
    validate: reportValidator(),
})