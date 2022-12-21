import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { RootPermission, SmartContract, SmartContractCreateInput, SmartContractSearchInput, SmartContractSortBy, SmartContractUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { SmartContractVersionModel } from "./smartContractVersion";
import { Displayer, ModelLogic } from "./types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: SmartContractCreateInput,
    GqlUpdate: SmartContractUpdateInput,
    GqlModel: SmartContract,
    GqlSearch: SmartContractSearchInput,
    GqlSort: SmartContractSortBy,
    GqlPermission: RootPermission,
    PrismaCreate: Prisma.smart_contractUpsertArgs['create'],
    PrismaUpdate: Prisma.smart_contractUpsertArgs['update'],
    PrismaModel: Prisma.smart_contractGetPayload<SelectWrap<Prisma.smart_contractSelect>>,
    PrismaSelect: Prisma.smart_contractSelect,
    PrismaWhere: Prisma.smart_contractWhereInput,
}

const __typename = 'SmartContract' as const;

const suppFields = [] as const;

const displayer = (): Displayer<Model> => ({
    select: () => ({
        id: true,
        versions: {
            orderBy: { versionIndex: 'desc' },
            take: 1,
            select: SmartContractVersionModel.display.select(),
        }
    }),
    label: (select, languages) => select.versions.length > 0 ?
        SmartContractVersionModel.display.label(select.versions[0] as any, languages) : '',
})

export const SmartContractModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.smart_contract,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})