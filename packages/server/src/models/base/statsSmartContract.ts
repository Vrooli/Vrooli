import { StatsSmartContractSortBy } from "@local/shared";
import { Prisma } from "@prisma/client";
import i18next from "i18next";
import { selPad } from "../../builders";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { StatsSmartContractFormat } from "../format/statsSmartContract";
import { ModelLogic } from "../types";
import { SmartContractModel } from "./smartContract";
import { StatsSmartContractModelLogic } from "./types";

const __typename = "StatsSmartContract" as const;
const suppFields = [] as const;
export const StatsSmartContractModel: ModelLogic<StatsSmartContractModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.stats_smart_contract,
    display: {
        label: {
            select: () => ({ id: true, smartContract: selPad(SmartContractModel.display.label.select) }),
            get: (select, languages) => i18next.t("common:ObjectStats", {
                lng: languages.length > 0 ? languages[0] : "en",
                objectName: SmartContractModel.display.label.get(select.smartContract as any, languages),
            }),
        },
    },
    format: StatsSmartContractFormat,
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