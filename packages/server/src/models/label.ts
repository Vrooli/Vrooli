import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Label, LabelCreateInput, LabelSearchInput, LabelSortBy, LabelUpdateInput, LabelYou, PrependString } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { getSingleTypePermissions } from "../validators";
import { defaultPermissions, oneIsPublic } from "../utils";
import { OrganizationModel } from "./organization";

const __typename = 'Label' as const;
type Permissions = Pick<LabelYou, 'canDelete' | 'canUpdate'>;
const suppFields = ['you'] as const;
export const LabelModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: LabelCreateInput,
    GqlUpdate: LabelUpdateInput,
    GqlModel: Label,
    GqlSearch: LabelSearchInput,
    GqlSort: LabelSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.labelUpsertArgs['create'],
    PrismaUpdate: Prisma.labelUpsertArgs['update'],
    PrismaModel: Prisma.labelGetPayload<SelectWrap<Prisma.labelSelect>>,
    PrismaSelect: Prisma.labelSelect,
    PrismaWhere: Prisma.labelWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.label,
    display: {
        select: () => ({ id: true, label: true }),
        label: (select) => select.label,
    },
    format: {
        gqlRelMap: {
            __typename,
            apis: 'Api',
            issues: 'Issue',
            meetings: 'Meeting',
            notes: 'Note',
            owner: {
                ownedByUser: 'User',
                ownedByOrganization: 'Organization',
            },
            projects: 'Project',
            routines: 'Routine',
            runProjectSchedules: 'RunProjectSchedule',
            runRoutineSchedules: 'RunRoutineSchedule',
            userSchedules: 'UserSchedule',
        },
        prismaRelMap: {
            __typename,
            apis: 'Api',
            issues: 'Issue',
            meetings: 'Meeting',
            notes: 'Note',
            ownedByUser: 'User',
            ownedByOrganization: 'Organization',
            projects: 'Project',
            routines: 'Routine',
            runProjectSchedules: 'RunProjectSchedule',
            runRoutineSchedules: 'RunRoutineSchedule',
            userSchedules: 'UserSchedule',
        },
        joinMap: {
            apis: 'labelled',
            issues: 'labelled',
            meetings: 'labelled',
            notes: 'labelled',
            projects: 'labelled',
            routines: 'labelled',
            runProjectSchedules: 'labelled',
            runRoutineSchedules: 'labelled',
            userSchedules: 'labelled',
        },
        countFields: {
            apisCount: true,
            issuesCount: true,
            meetingsCount: true,
            notesCount: true,
            projectsCount: true,
            smartContractsCount: true,
            standardsCount: true,
            routinesCount: true,
            runProjectSchedulesCount: true,
            runRoutineSchedulesCount: true,
            translationsCount: true,
            userSchedulesCount: true,
        },
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                    }
                }
            },
        },
    },
    mutate: {} as any,
    search: {
        defaultSort: LabelSortBy.DateUpdatedDesc,
        sortBy: LabelSortBy,
        searchFields: {
            createdTimeFrame: true,
            ownedByOrganizationId: true,
            ownedByUserId: true,
            translationLanguages: true,
            updatedTimeFrame: true,
            visibility: true,
        },
        searchStringQuery: () => ({
            OR: [
                'labelWrapped',
                'transDescriptionWrapped'
            ]
        })
    },
    validate: {
        isTransferable: false,
        maxObjects: {
            User: {
                private: {
                    noPremium: 20,
                    premium: 100,
                },
                public: {
                    noPremium: 20,
                    premium: 100,
                }
            },
            Organization: {
                private: {
                    noPremium: 20,
                    premium: 100,
                },
                public: {
                    noPremium: 20,
                    premium: 100,
                },
            }
        },
        permissionsSelect: () => ({
            id: true,
            ownedByOrganization: 'Organization',
            ownedByUser: 'User',
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => ({
            Organization: data.ownedByOrganization,
            User: data.ownedByUser,
        }),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.commentSelect>(data, [
            ['ownedByOrganization', 'Organization'],
            ['ownedByUser', 'User'],
        ], languages),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                OR: [
                    { ownedByUser: { id: userId } },
                    { ownedByOrganization: OrganizationModel.query.hasRoleQuery(userId) },
                ]
            }),
        }
    },
})