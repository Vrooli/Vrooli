import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrependString, RunProject, RunProjectCreateInput, RunProjectSearchInput, RunProjectSortBy, RunProjectUpdateInput, RunProjectYou } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { getSingleTypePermissions } from "../validators";

const __typename = 'RunProject' as const;
type Permissions = Pick<RunProjectYou, 'canDelete' | 'canEdit' | 'canView'>;
const suppFields = ['you.canDelete', 'you.canEdit', 'you.canView'] as const;
export const RunProjectModel: ModelLogic<{
    IsTransferable: true,
    IsVersioned: true,
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
                let permissions = await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData);
                return {
                    ...(Object.fromEntries(Object.entries(permissions).map(([k, v]) => [`you.${k}`, v])) as PrependString<typeof permissions, 'you.'>),
                }
            },
        },
    },
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})