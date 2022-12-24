import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { RunProject, RunProjectCreateInput, RunProjectPermission, RunProjectSearchInput, RunProjectSortBy, RunProjectUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { Displayer, ModelLogic } from "./types";

type Model = {
    IsTransferable: true,
    IsVersioned: true,
    GqlCreate: RunProjectCreateInput,
    GqlUpdate: RunProjectUpdateInput,
    GqlModel: RunProject,
    GqlSearch: RunProjectSearchInput,
    GqlSort: RunProjectSortBy,
    GqlPermission: RunProjectPermission,
    PrismaCreate: Prisma.run_projectUpsertArgs['create'],
    PrismaUpdate: Prisma.run_projectUpsertArgs['update'],
    PrismaModel: Prisma.run_projectGetPayload<SelectWrap<Prisma.run_projectSelect>>,
    PrismaSelect: Prisma.run_projectSelect,
    PrismaWhere: Prisma.run_projectWhereInput,
}

const __typename = 'RunProject' as const;

const suppFields = [] as const;

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, name: true }),
    label: (select) => select.name,
})

export const RunProjectModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.run_project,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})