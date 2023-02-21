import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { RunProjectStep, RunProjectStepCreateInput, RunProjectStepUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";

const __typename = 'RunProjectStep' as const;
const suppFields = [] as const;
export const RunProjectStepModel: ModelLogic<{
    IsTransferable: true,
    IsVersioned: true,
    GqlCreate: RunProjectStepCreateInput,
    GqlUpdate: RunProjectStepUpdateInput,
    GqlModel: RunProjectStep,
    GqlPermission: {},
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.run_project_stepUpsertArgs['create'],
    PrismaUpdate: Prisma.run_project_stepUpsertArgs['update'],
    PrismaModel: Prisma.run_project_stepGetPayload<SelectWrap<Prisma.run_project_stepSelect>>,
    PrismaSelect: Prisma.run_project_stepSelect,
    PrismaWhere: Prisma.run_project_stepWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.run_project_step,
    display: {
        select: () => ({ id: true, name: true }),
        label: (select) => select.name,
    },
    format: {
        gqlRelMap: {
            __typename,
            directory: 'ProjectVersionDirectory',
            run: 'RunProject',
        },
        prismaRelMap: {
            __typename,
            directory: 'ProjectVersionDirectory',
            runProject: 'RunProject',
        },
        countFields: {},
    },
    mutate: {} as any,
    validate: {} as any,
})