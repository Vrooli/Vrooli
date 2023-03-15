import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Label, LabelCreateInput, LabelSearchInput, LabelSortBy, LabelUpdateInput, LabelYou, MaxObjects, PrependString } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { getSingleTypePermissions } from "../validators";
import { defaultPermissions, oneIsPublic, translationShapeHelper } from "../utils";
import { OrganizationModel } from "./organization";
import { labelValidation } from "@shared/validation";
import { noNull, shapeHelper } from "../builders";

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
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                label: data.label,
                color: noNull(data.color),
                ownedByOrganization: data.organizationConnect ? { connect: { id: data.organizationConnect } } : undefined,
                ownedByUser: !data.organizationConnect ? { connect: { id: rest.userData.id } } : undefined,
                ...(await translationShapeHelper({ relTypes: ['Create'], isRequired: false, data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                id: data.id,
                label: noNull(data.label),
                color: noNull(data.color),
                ...(await shapeHelper({ relation: 'apis', relTypes: ['Connect', 'Disconnect'], isOneToOne: false, isRequired: false, objectType: 'Api', parentRelationshipName: 'issues', data, ...rest })),
                ...(await shapeHelper({ relation: 'issues', relTypes: ['Connect', 'Disconnect'], isOneToOne: false, isRequired: false, objectType: 'Issue', parentRelationshipName: 'issues', data, ...rest })),
                ...(await shapeHelper({ relation: 'meetings', relTypes: ['Connect', 'Disconnect'], isOneToOne: false, isRequired: false, objectType: 'Meeting', parentRelationshipName: 'issues', data, ...rest })),
                ...(await shapeHelper({ relation: 'notes', relTypes: ['Connect', 'Disconnect'], isOneToOne: false, isRequired: false, objectType: 'Note', parentRelationshipName: 'issues', data, ...rest })),
                ...(await shapeHelper({ relation: 'projects', relTypes: ['Connect', 'Disconnect'], isOneToOne: false, isRequired: false, objectType: 'Project', parentRelationshipName: 'issues', data, ...rest })),
                ...(await shapeHelper({ relation: 'routines', relTypes: ['Connect', 'Disconnect'], isOneToOne: false, isRequired: false, objectType: 'Routine', parentRelationshipName: 'issues', data, ...rest })),
                ...(await shapeHelper({ relation: 'runProjectSchedules', relTypes: ['Connect', 'Disconnect'], isOneToOne: false, isRequired: false, objectType: 'RunProjectSchedule', parentRelationshipName: 'issues', data, ...rest })),
                ...(await shapeHelper({ relation: 'runRoutineSchedules', relTypes: ['Connect', 'Disconnect'], isOneToOne: false, isRequired: false, objectType: 'RunRoutineSchedule', parentRelationshipName: 'issues', data, ...rest })),
                ...(await shapeHelper({ relation: 'smartContracts', relTypes: ['Connect', 'Disconnect'], isOneToOne: false, isRequired: false, objectType: 'SmartContract', parentRelationshipName: 'issues', data, ...rest })),
                ...(await shapeHelper({ relation: 'standards', relTypes: ['Connect', 'Disconnect'], isOneToOne: false, isRequired: false, objectType: 'Standard', parentRelationshipName: 'issues', data, ...rest })),
                ...(await shapeHelper({ relation: 'userSchedules', relTypes: ['Connect', 'Disconnect'], isOneToOne: false, isRequired: false, objectType: 'UserSchedule', parentRelationshipName: 'issues', data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ['Create', 'Update', 'Delete'], isRequired: false, data, ...rest })),
            })
        },
        yup: labelValidation,
    },
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
        maxObjects: MaxObjects[__typename],
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