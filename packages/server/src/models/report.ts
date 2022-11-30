import { reportsCreate, reportsUpdate } from "@shared/validation";
import { ReportFor, ReportSortBy } from '@shared/consts';
import { Report, ReportSearchInput, ReportCreateInput, ReportUpdateInput } from "../schema/types";
import { PrismaType } from "../types";
import { Formatter, Searcher, GraphQLModelType, Validator, Mutater, Displayer } from "./types";
import { Prisma, ReportStatus } from "@prisma/client";
import { CustomError, Trigger } from "../events";
import { UserModel } from "./user";
import { combineQueries, padSelect } from "../builders";
import { CommentModel } from "./comment";
import { OrganizationModel } from "./organization";
import { ProjectModel } from "./project";
import { RoutineModel } from "./routine";
import { StandardModel } from "./standard";
import { TagModel } from "./tag";

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
    Prisma.reportGetPayload<{ select: { [K in keyof Required<Prisma.reportSelect>]: true } }>,
    any,
    Prisma.reportSelect,
    Prisma.reportWhereInput,
    false,
    false
> => ({
    validateMap: {
        __typename: 'Report',
    },
    isTransferable: false,
    maxObjects: {
        User: {
            private: 0,
            public: 10000,
        },
        Organization: 0,
    },
    permissionsSelect: (...params) => ({ id: true, createdBy: { select: UserModel.validate.permissionsSelect(...params) } }),
    permissionResolvers: ({ isAdmin }) => ([
        ['isOwn', async () => isAdmin],
    ]),
    owner: (data) => ({
        User: data.createdBy,
    }),
    isDeleted: () => false,
    isPublic: () => true,
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
    visibility: {
        private: {},
        public: {},
        owner: (userId) => ({ createdBy: { id: userId } }),
    }
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

const displayer = (): Displayer<
    Prisma.reportSelect,
    Prisma.reportGetPayload<{ select: { [K in keyof Required<Prisma.reportSelect>]: true } }>
> => ({
    select: () => ({
        id: true,
        // apiVersion: padSelect(ApiModel.display.select),
        comment: padSelect(CommentModel.display.select),
        // issue: padSelect(IssueModel.display.select),
        organization: padSelect(OrganizationModel.display.select),
        // post: padSelect(PostModel.display.select),
        projectVersion: padSelect(ProjectModel.display.select),
        routineVersion: padSelect(RoutineModel.display.select),
        // smartContractVersion: padSelect(SmartContractModel.display.select),
        standardVersion: padSelect(StandardModel.display.select),
        tag: padSelect(TagModel.display.select),
        user: padSelect(UserModel.display.select),
    }),
    label: (select, languages) => {
        // if (select.apiVersion) return ApiModel.display.label(select.api as any, languages);
        if (select.comment) return CommentModel.display.label(select.comment as any, languages);
        // if (select.issue) return IssueModel.display.label(select.issue as any, languages);
        if (select.organization) return OrganizationModel.display.label(select.organization as any, languages);
        // if (select.post) return PostModel.display.label(select.post as any, languages);
        if (select.projectVersion) return ProjectModel.display.label(select.projectVersion as any, languages);
        if (select.routineVersion) return RoutineModel.display.label(select.routineVersion as any, languages);
        // if (select.smartContractVersion) return SmartContractModel.display.label(select.smartContractVersion as any, languages);
        if (select.standardVersion) return StandardModel.display.label(select.standardVersion as any, languages);
        if (select.tag) return TagModel.display.label(select.tag as any, languages);
        if (select.user) return UserModel.display.label(select.user as any, languages);
        return '';
    }
})

export const ReportModel = ({
    delegate: (prisma: PrismaType) => prisma.report,
    display: displayer(),
    format: formatter(),
    mutate: mutater(),
    search: searcher(),
    type: 'Report' as GraphQLModelType,
    validate: validator(),
})