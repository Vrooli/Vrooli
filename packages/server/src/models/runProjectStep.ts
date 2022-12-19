import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { RunProjectStep, RunProjectStepCreateInput, RunProjectStepPermission, RunProjectStepSearchInput, RunProjectStepSortBy, RunProjectStepUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { Displayer, GraphQLModelType, ModelLogic } from "./types";

type Model = {
    IsTransferable: true,
    IsVersioned: true,
    GqlCreate: RunProjectStepCreateInput,
    GqlUpdate: RunProjectStepUpdateInput,
    GqlModel: RunProjectStep,
    GqlSearch: RunProjectStepSearchInput,
    GqlSort: RunProjectStepSortBy,
    GqlPermission: RunProjectStepPermission,
    PrismaCreate: Prisma.run_project_stepUpsertArgs['create'],
    PrismaUpdate: Prisma.run_project_stepUpsertArgs['update'],
    PrismaModel: Prisma.run_project_stepGetPayload<SelectWrap<Prisma.run_project_stepSelect>>,
    PrismaSelect: Prisma.run_project_stepSelect,
    PrismaWhere: Prisma.run_project_stepWhereInput,
}

const __typename = 'RunProjectStep' as const;

const suppFields = [] as const;

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, name: true }),
    label: (select) => select.name,
})

export const RunProjectStepModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.run_project_step,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})