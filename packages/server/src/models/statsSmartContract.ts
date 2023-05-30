import { StatsSmartContract, StatsSmartContractSearchInput, StatsSmartContractSortBy } from "@local/shared";
import { Prisma } from "@prisma/client";
import i18next from "i18next";
import { selPad } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { defaultPermissions, oneIsPublic } from "../utils";
import { SmartContractModel } from "./smartContract";
import { ModelLogic } from "./types";

const __typename = "StatsSmartContract" as const;
const suppFields = [] as const;
export const StatsSmartContractModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: undefined,
    GqlUpdate: undefined,
    GqlModel: StatsSmartContract,
    GqlSearch: StatsSmartContractSearchInput,
    GqlSort: StatsSmartContractSortBy,
    GqlPermission: object,
    PrismaCreate: Prisma.stats_smart_contractUpsertArgs["create"],
    PrismaUpdate: Prisma.stats_smart_contractUpsertArgs["update"],
    PrismaModel: Prisma.stats_smart_contractGetPayload<SelectWrap<Prisma.stats_smart_contractSelect>>,
    PrismaSelect: Prisma.stats_smart_contractSelect,
    PrismaWhere: Prisma.stats_smart_contractWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.stats_smart_contract,
    display: {
        label: {
            select: () => ({ id: true, smartContract: selPad(SmartContractModel.display.label.select) }),
            get: (select, languages) => i18next.t("common:ObjectStats", {
                lng: languages.length > 0 ? languages[0] : "en",
                objectName: SmartContractModel.display.label.get(select.smartContract as any, languages),
            }),
        },
    },
    format: {
        gqlRelMap: {
            __typename,
        },
        prismaRelMap: {
            __typename,
            smartContract: "SmartContract",
        },
        countFields: {},
    },
    search: {
        defaultSort: StatsSmartContractSortBy.PeriodStartAsc,
        sortBy: StatsSmartContractSortBy,
        searchFields: {
            periodTimeFrame: true,
            periodType: true,
        },
        searchStringQuery: () => ({ smartContract: SmartContractModel.search!.searchStringQuery() }),
    },
    validate: {
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            smartContract: "SmartContract",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => SmartContractModel.validate!.owner(data.smartContract as any, userId),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.stats_smart_contractSelect>(data, [
            ["smartContract", "SmartContract"],
        ], languages),
        visibility: {
            private: { smartContract: SmartContractModel.validate!.visibility.private },
            public: { smartContract: SmartContractModel.validate!.visibility.public },
            owner: (userId) => ({ smartContract: SmartContractModel.validate!.visibility.owner(userId) }),
        },
    },
});
