import { StatsSmartContract, StatsSmartContractSearchInput, StatsSmartContractSortBy } from "@local/shared";
import { Prisma } from "@prisma/client";
import i18next from "i18next";
import { selPad } from "../../builders";
import { SelectWrap } from "../../builders/types";
import { PrismaType } from "../../types";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { SmartContractModel } from "./smartContract";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "StatsSmartContract" as const;
export const StatsSmartContractFormat: Formatter<ModelStatsSmartContractLogic> = {
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
};
