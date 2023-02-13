import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { RunProject, RunProjectCreateInput, RunProjectSearchInput, RunProjectSortBy, RunProjectUpdateInput, RunProjectYou } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { getSingleTypePermissions } from "../validators";
import { OrganizationModel } from "./organization";
import { defaultPermissions, oneIsPublic } from "../utils";

const __typename = 'RunProject' as const;
type Permissions = Pick<RunProjectYou, 'canDelete' | 'canUpdate' | 'canRead'>;
const suppFields = ['you'] as const;
export const RunProjectModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: RunProjectCreateInput,
    GqlUpdate: RunProjectUpdateInput,
    GqlModel: RunProject,
    GqlSearch: RunProjectSearchInput,
    GqlSort: RunProjectSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.run_projectUpsertArgs['create'],
    PrismaUpdate: Prisma.run_projectUpsertArgs['update'],
    PrismaModel: Prisma.run_projectGetPayload<SelectWrap<Prisma.run_projectSelect>>,
    PrismaSelect: Prisma.run_projectSelect,
    PrismaWhere: Prisma.run_projectWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.run_project,
    display: {
        select: () => ({ id: true, name: true }),
        label: (select) => select.name,
    },
    format: {
        gqlRelMap: {
            __typename,
            projectVersion: 'ProjectVersion',
            runProjectSchedule: 'RunProjectSchedule',
            steps: 'RunProjectStep',
            user: 'User',
            organization: 'Organization',
        },
        prismaRelMap: {
            __typename,
            projectVersion: 'ProjectVersion',
            runProjectSchedule: 'RunProjectSchedule',
            steps: 'RunProjectStep',
            user: 'User',
            organization: 'Organization',
        },
        countFields: {
            stepsCount: true,
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
    search: {} as any,
    validate: {
        isTransferable: false,
        maxObjects: {
            User: 5000,
            Organization: 50000,
        },
        permissionsSelect: () => ({
            id: true,
            isPrivate: true,
            organization: 'Organization',
            projectVersion: 'Routine',
            user: 'User',
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => ({
            Organization: data.organization,
            User: data.user,
        }),
        isDeleted: () => false,
        isPublic: (data, languages,) => data.isPrivate === false && oneIsPublic<Prisma.run_projectSelect>(data, [
            ['organization', 'Organization'],
            ['user', 'User'],
        ], languages),
        profanityFields: ['name'],
        validations: {
            async update({ languages, updateMany }) {
                // TODO if status passed in for update, make sure it's not the same 
                // as the current status, or an invalid transition (e.g. failed -> in progress)
            },
        },
        visibility: {
            private: { isPrivate: true },
            public: { isPrivate: false },
            owner: (userId) => ({
                OR: [
                    { user: { id: userId } },
                    { organization: OrganizationModel.query.hasRoleQuery(userId) },
                ]
            }),
        },
    },
})