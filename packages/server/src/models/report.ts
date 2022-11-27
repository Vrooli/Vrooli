import { reportsCreate, reportsUpdate } from "@shared/validation";
import { ReportFor, ReportSortBy } from '@shared/consts';
import { Report, ReportSearchInput, ReportCreateInput, ReportUpdateInput } from "../schema/types";
import { PrismaType } from "../types";
import { Formatter, Searcher, GraphQLModelType, Validator, Mutater } from "./types";
import { Prisma, ReportStatus } from "@prisma/client";
import { CustomError, Trigger } from "../events";
import { UserModel } from "./user";
import { combineQueries } from "../builders";

type SupplementalFields = 'isOwn';
const formatter = (): Formatter<Report, SupplementalFields> => ({
    relationshipMap: { __typename: 'Report' },
    hiddenFields: ['userId'], // Always hide report creator
    supplemental: {
        graphqlFields: ['isOwn'],
        dbFields: ['userId'],
        toGraphQL: ({ objects, userData }) => [
            ['isOwn', async () => objects.map((x) => Boolean(userData) && x.fromId === userData?.id)],
        ],
    },
})

const searcher = (): Searcher<
    ReportSearchInput,
    ReportSortBy,
    Prisma.reportOrderByWithRelationInput,
    Prisma.reportWhereInput
> => ({
    defaultSort: ReportSortBy.DateCreatedDesc,
    sortMap: {
        DateCreatedAsc: { created_at: 'asc' },
        DateCreatedDesc: { created_at: 'desc' },
    },
    searchStringQuery: ({ insensitive }) => ({
        OR: [
            { reason: { ...insensitive } },
            { details: { ...insensitive } },
        ]
    }),
    customQueries(input) {
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

const forMapper: { [key in ReportFor]: keyof Prisma.reportUpsertArgs['create'] } = {
    Comment: 'comment',
    Organization: 'organization',
    Project: 'projectVersion',
    Routine: 'routineVersion',
    Standard: 'standardVersion',
    Tag: 'tag',
    User: 'user',
}

const validator = (): Validator<
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
    },
    isTransferable: false,
    permissionsSelect: (...params) => ({ id: true, createdBy: { select: UserModel.validate.permissionsSelect(...params) } }),
    permissionResolvers: ({ isAdmin }) => ([
        ['isOwn', async () => isAdmin],
    ]),
    owner: (data) => ({
        User: data.createdBy,
    }),
    isDeleted: () => false,
    isPublic: () => true,
    ownerOrMemberWhere: (userId) => ({ createdById: userId }),
    profanityFields: ['reason', 'details'],
    validations: {
        create: async ({ createMany, prisma, userData }) => {
            // Make sure user does not have any open reports on these objects
            const existing = await prisma.report.findMany({
                where: {
                    status: 'Open',
                    user: { id: userData.id },
                    OR: createMany.map((x) => ({
                        [`${forMapper[x.createdFor]}Id`]: { id: x.createdForId },
                    })),
                },
            });
            if (existing.length > 0)
                throw new CustomError('0337', 'MaxReportsReached', userData.languages);
        }
    },
})

const mutater = (): Mutater<
    Report,
    { graphql: ReportCreateInput, db: Prisma.reportUpsertArgs['create'] },
    { graphql: ReportUpdateInput, db: Prisma.reportUpsertArgs['update'] },
    false,
    false
> => ({
    shape: {
        create: async ({ data, userData }) => {
            return {
                id: data.id,
                language: data.language,
                reason: data.reason,
                details: data.details,
                status: ReportStatus.Open,
                from: { connect: { id: userData.id } },
                [forMapper[data.createdFor]]: { connect: { id: data.createdForId } },
            }
        },
        update: async ({ data }) => {
            return {
                reason: data.reason ?? undefined,
                details: data.details,
            }
        }
    },
    trigger: {
        onCreated: ({ created, prisma, userData }) => {
            for (const c of created) {
                Trigger(prisma, userData.languages).reportOpen(c.id as string, userData.id);
            }
        },
    },
    yup: { create: reportsCreate, update: reportsUpdate },
})

export const ReportModel = ({
    delegate: (prisma: PrismaType) => prisma.report,
    format: formatter(),
    mutate: mutater(),
    search: searcher(),
    type: 'Report' as GraphQLModelType,
    validate: validator(),
})