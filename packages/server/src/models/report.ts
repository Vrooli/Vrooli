import { reportValidation } from "@shared/validation";
import { PrependString, ReportFor, ReportSortBy, ReportYou } from '@shared/consts';
import { Report, ReportSearchInput, ReportCreateInput, ReportUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { Prisma, ReportStatus } from "@prisma/client";
import { CustomError, Trigger } from "../events";
import { UserModel } from "./user";
import { padSelect } from "../builders";
import { CommentModel } from "./comment";
import { OrganizationModel } from "./organization";
import { TagModel } from "./tag";
import { SelectWrap } from "../builders/types";
import { ApiVersionModel } from "./apiVersion";
import { IssueModel } from "./issue";
import { NoteVersionModel } from "./noteVersion";
import { PostModel } from "./post";
import { ProjectVersionModel } from "./projectVersion";
import { RoutineVersionModel } from "./routineVersion";
import { SmartContractVersionModel } from "./smartContractVersion";
import { StandardVersionModel } from "./standardVersion";
import { getSingleTypePermissions } from "../validators";

const forMapper: { [key in ReportFor]: keyof Prisma.reportUpsertArgs['create'] } = {
    ApiVersion: "apiVersion",
    Comment: "comment",
    Issue: "issue",
    Organization: "organization",
    NoteVersion: "noteVersion",
    Post: "post",
    ProjectVersion: "projectVersion",
    RoutineVersion: "routineVersion",
    StandardVersion: "standardVersion",
    Tag: "tag",
    User: "user",
}

const type = 'Report' as const;
type Permissions = Pick<ReportYou, 'canDelete' | 'canEdit' | 'canRespond'>;
const suppFields = ['you.canDelete', 'you.canEdit', 'you.canRespond'] as const;
export const ReportModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: ReportCreateInput,
    GqlUpdate: ReportUpdateInput,
    GqlModel: Report,
    GqlSearch: ReportSearchInput,
    GqlSort: ReportSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.reportUpsertArgs['create'],
    PrismaUpdate: Prisma.reportUpsertArgs['update'],
    PrismaModel: Prisma.reportGetPayload<SelectWrap<Prisma.reportSelect>>,
    PrismaSelect: Prisma.reportSelect,
    PrismaWhere: Prisma.reportWhereInput,
}, typeof suppFields> = ({
    type,
    delegate: (prisma: PrismaType) => prisma.report,
    display: {
        select: () => ({
            id: true,
            apiVersion: padSelect(ApiVersionModel.display.select),
            comment: padSelect(CommentModel.display.select),
            issue: padSelect(IssueModel.display.select),
            noteVersion: padSelect(NoteVersionModel.display.select),
            organization: padSelect(OrganizationModel.display.select),
            post: padSelect(PostModel.display.select),
            projectVersion: padSelect(ProjectVersionModel.display.select),
            routineVersion: padSelect(RoutineVersionModel.display.select),
            smartContractVersion: padSelect(SmartContractVersionModel.display.select),
            standardVersion: padSelect(StandardVersionModel.display.select),
            tag: padSelect(TagModel.display.select),
            user: padSelect(UserModel.display.select),
        }),
        label: (select, languages) => {
            if (select.apiVersion) return ApiVersionModel.display.label(select.apiVersion as any, languages);
            if (select.comment) return CommentModel.display.label(select.comment as any, languages);
            if (select.issue) return IssueModel.display.label(select.issue as any, languages);
            if (select.noteVersion) return NoteVersionModel.display.label(select.noteVersion as any, languages);
            if (select.organization) return OrganizationModel.display.label(select.organization as any, languages);
            if (select.post) return PostModel.display.label(select.post as any, languages);
            if (select.projectVersion) return ProjectVersionModel.display.label(select.projectVersion as any, languages);
            if (select.routineVersion) return RoutineVersionModel.display.label(select.routineVersion as any, languages);
            if (select.smartContractVersion) return SmartContractVersionModel.display.label(select.smartContractVersion as any, languages);
            if (select.standardVersion) return StandardVersionModel.display.label(select.standardVersion as any, languages);
            if (select.tag) return TagModel.display.label(select.tag as any, languages);
            if (select.user) return UserModel.display.label(select.user as any, languages);
            return '';
        }
    },
    format: {
        gqlRelMap: { type },
        prismaRelMap: { type },
        hiddenFields: ['userId'], // Always hide report creator
        supplemental: {
            graphqlFields: suppFields,
            dbFields: ['userId'],
            toGraphQL: async ({ ids, prisma, userData }) => {
                let permissions = await getSingleTypePermissions<Permissions>(type, ids, prisma, userData);
                return Object.fromEntries(Object.entries(permissions).map(([k, v]) => [`you.${k}`, v])) as PrependString<typeof permissions, 'you.'>
            },
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, userData }) => {
                return {
                    id: data.id,
                    language: data.language,
                    reason: data.reason,
                    details: data.details,
                    status: ReportStatus.Open,
                    createdBy: { connect: { id: userData.id } },
                    [forMapper[data.createdFor]]: { connect: { id: data.createdForConnect } },
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
        yup: reportValidation,
    },
    search: {
        defaultSort: ReportSortBy.DateCreatedDesc,
        sortBy: ReportSortBy,
        searchFields: {
            apiVersionId: true,
            commentId: true,
            createdTimeFrame: true,
            fromId: true,
            issueId: true,
            languageIn: true,
            noteVersionId: true,
            organizationId: true,
            postId: true,
            projectVersionId: true,
            routineVersionId: true,
            smartContractVersionId: true,
            standardVersionId: true,
            tagId: true,
            updatedTimeFrame: true,
            userId: true,
        },
        searchStringQuery: () => ({
            OR: [
                'detailsWrapped',
                'reasonWrapped',
            ]
        }),
    },
    validate: {
        isTransferable: false,
        maxObjects: {
            User: {
                private: 0,
                public: 10000,
            },
            Organization: 0,
        },
        permissionsSelect: (...params) => ({
            id: true,
            createdBy: 'User',
        }),
        permissionResolvers: ({ data, isAdmin }) => ({
            canDelete: () => isAdmin && data.status !== 'Open',
            canEdit: () => isAdmin && data.status !== 'Open',
            canRespond: () => data.status === 'Open',
        }),
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
                            [`${forMapper[x.createdFor]}Id`]: { id: x.createdForConnect },
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
    },
})