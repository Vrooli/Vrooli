import { StatsSmartContractSortBy } from "@local/shared";
import i18next from "i18next";
import { ModelMap } from ".";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { StatsSmartContractFormat } from "../formats";
import { SmartContractModelInfo, SmartContractModelLogic, StatsSmartContractModelInfo, StatsSmartContractModelLogic } from "./types";

const __typename = "StatsSmartContract" as const;
export const StatsSmartContractModel: StatsSmartContractModelLogic = ({
    __typename,
    dbTable: "stats_smart_contract",
    display: () => ({
        label: {
            select: () => ({ id: true, smartContract: { select: ModelMap.get<SmartContractModelLogic>("SmartContract").display().label.select() } }),
            get: (select, languages) => i18next.t("common:ObjectStats", {
                lng: languages.length > 0 ? languages[0] : "en",
                objectName: ModelMap.get<SmartContractModelLogic>("SmartContract").display().label.get(select.smartContract as SmartContractModelInfo["PrismaModel"], languages),
            }),
        },
    }),
    format: StatsSmartContractFormat,
    search: {
        defaultSort: StatsSmartContractSortBy.PeriodStartAsc,
        sortBy: StatsSmartContractSortBy,
        searchFields: {
            periodTimeFrame: true,
            periodType: true,
        },
        searchStringQuery: () => ({ smartContract: ModelMap.get<SmartContractModelLogic>("SmartContract").search.searchStringQuery() }),
    },
    validate: () => ({
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            smartContract: "SmartContract",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<SmartContractModelLogic>("SmartContract").validate().owner(data?.smartContract as SmartContractModelInfo["PrismaModel"], userId),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<StatsSmartContractModelInfo["PrismaSelect"]>([["smartContract", "SmartContract"]], ...rest),
        visibility: {
            private: { smartContract: ModelMap.get<SmartContractModelLogic>("SmartContract").validate().visibility.private },
            public: { smartContract: ModelMap.get<SmartContractModelLogic>("SmartContract").validate().visibility.public },
            owner: (userId) => ({ smartContract: ModelMap.get<SmartContractModelLogic>("SmartContract").validate().visibility.owner(userId) }),
        },
    }),
});
