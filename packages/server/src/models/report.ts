import { reportsCreate, reportsUpdate } from "@shared/validation";
import { ReportFor, ReportSortBy } from '@shared/consts';
import { combineQueries, getSearchStringQueryHelper } from "./builder";
import { Report, ReportSearchInput, ReportCreateInput, ReportUpdateInput } from "../schema/types";
import { PrismaType } from "../types";
import { FormatConverter, Searcher, CUDInput, CUDResult, GraphQLModelType, Validator, Mutater } from "./types";
import { Prisma, ReportStatus } from "@prisma/client";
import { cudHelper } from "./actions";
import { Trigger } from "../events";

type SupplementalFields = 'isOwn';
export const reportFormatter = (): FormatConverter<Report, SupplementalFields> => ({
    relationshipMap: { __typename: 'Report' },
    removeJoinTables: (data) => {
        // Remove userId to hide who submitted the report
        let { userId, ...rest } = data;
        return rest;
    },
    supplemental: {
        graphqlFields: ['isOwn'],
        dbFields: ['userId'],
        toGraphQL: ({ objects, userId }) => [
            ['isOwn', async () => objects.map((x) => Boolean(userId) && x.fromId === userId)],
        ],
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

export const reportValidator = (): Validator<
    ReportCreateInput,
    ReportUpdateInput,
    Report,
    Prisma.reportGetPayload<{ select: { [K in keyof Required<Prisma.reportSelect>]: true } }>,
    any,
    Prisma.reportSelect,
    Prisma.reportWhereInput
> => ({
    validateMap: {
        __typename: 'Report',
        asdfasdf
    },
    permissionsSelect: {
        id: true,
        user: { select: { id: true } },
    },
    permissionsFromSelect: (select, userId) => asdf as any,
    isPublic: () => true,
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

const forMapper: { [key in ReportFor]: string } = {
    Comment: 'comment',
    Organization: 'organization',
    Project: 'project',
    Routine: 'routine',
    Standard: 'standard',
    Tag: 'tag',
    User: 'user',
}

export const reportMutater = (prisma: PrismaType): Mutater<Report> => ({
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
            yup: { yupCreate: reportsCreate, yupUpdate: reportsUpdate },
            shape: { shapeCreate: this.shapeCreate, shapeUpdate: this.shapeUpdate },
            onCreated: (created) => {
                for (const c of created) {
                    Trigger(prisma).objectCreate('Report', c.id as string, params.userId);
                }
            },
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