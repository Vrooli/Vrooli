import { StatsSmartContractSortBy } from "@local/consts";
import i18next from "i18next";
import { selPad } from "../builders";
import { defaultPermissions, oneIsPublic } from "../utils";
import { SmartContractModel } from "./smartContract";
const __typename = "StatsSmartContract";
const suppFields = [];
export const StatsSmartContractModel = ({
    __typename,
    delegate: (prisma) => prisma.stats_smart_contract,
    display: {
        select: () => ({ id: true, smartContract: selPad(SmartContractModel.display.select) }),
        label: (select, languages) => i18next.t("common:ObjectStats", {
            lng: languages.length > 0 ? languages[0] : "en",
            objectName: SmartContractModel.display.label(select.smartContract, languages),
        }),
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
        searchStringQuery: () => ({ smartContract: SmartContractModel.search.searchStringQuery() }),
    },
    validate: {
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            smartContract: "SmartContract",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => SmartContractModel.validate.owner(data.smartContract, userId),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic(data, [
            ["smartContract", "SmartContract"],
        ], languages),
        visibility: {
            private: { smartContract: SmartContractModel.validate.visibility.private },
            public: { smartContract: SmartContractModel.validate.visibility.public },
            owner: (userId) => ({ smartContract: SmartContractModel.validate.visibility.owner(userId) }),
        },
    },
});
//# sourceMappingURL=statsSmartContract.js.map