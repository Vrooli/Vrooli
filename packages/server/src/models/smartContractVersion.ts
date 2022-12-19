import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { SmartContractVersion, SmartContractVersionCreateInput, SmartContractVersionPermission, SmartContractVersionSearchInput, SmartContractVersionSortBy, SmartContractVersionUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { Displayer, ModelLogic } from "./types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: SmartContractVersionCreateInput,
    GqlUpdate: SmartContractVersionUpdateInput,
    GqlModel: SmartContractVersion,
    GqlSearch: SmartContractVersionSearchInput,
    GqlSort: SmartContractVersionSortBy,
    GqlPermission: SmartContractVersionPermission,
    PrismaCreate: Prisma.smart_contract_versionUpsertArgs['create'],
    PrismaUpdate: Prisma.smart_contract_versionUpsertArgs['update'],
    PrismaModel: Prisma.smart_contract_versionGetPayload<SelectWrap<Prisma.smart_contract_versionSelect>>,
    PrismaSelect: Prisma.smart_contract_versionSelect,
    PrismaWhere: Prisma.smart_contract_versionWhereInput,
}

const __typename = 'SmartContractVersion' as const;

const suppFields = [] as const;

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, callLink: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'name', languages)
})

export const SmartContractVersionModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.smart_contract_version,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})