import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { RootPermission, SmartContract, SmartContractCreateInput, SmartContractSearchInput, SmartContractSortBy, SmartContractUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { SmartContractVersionModel } from "./smartContractVersion";
import { ModelLogic } from "./types";

const __typename = 'SmartContract' as const;
const suppFields = ['isStarred', 'isUpvoted', 'isViewed', 'permissionsRoot', 'translatedName'] as const;
export const SmartContractModel: ModelLogic<{
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
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.smart_contract,
    display: {
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
    },
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})