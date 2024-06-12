import { StatsCodeSortBy } from "@local/shared";
import i18next from "i18next";
import { ModelMap } from ".";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { StatsCodeFormat } from "../formats";
import { CodeModelInfo, CodeModelLogic, StatsCodeModelInfo, StatsCodeModelLogic } from "./types";

const __typename = "StatsCode" as const;
export const StatsCodeModel: StatsCodeModelLogic = ({
    __typename,
    dbTable: "stats_code",
    display: () => ({
        label: {
            select: () => ({ id: true, code: { select: ModelMap.get<CodeModelLogic>("Code").display().label.select() } }),
            get: (select, languages) => i18next.t("common:ObjectStats", {
                lng: languages.length > 0 ? languages[0] : "en",
                objectName: ModelMap.get<CodeModelLogic>("Code").display().label.get(select.code as CodeModelInfo["PrismaModel"], languages),
            }),
        },
    }),
    format: StatsCodeFormat,
    search: {
        defaultSort: StatsCodeSortBy.PeriodStartAsc,
        sortBy: StatsCodeSortBy,
        searchFields: {
            periodTimeFrame: true,
            periodType: true,
        },
        searchStringQuery: () => ({ code: ModelMap.get<CodeModelLogic>("Code").search.searchStringQuery() }),
    },
    validate: () => ({
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            code: "Code",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<CodeModelLogic>("Code").validate().owner(data?.code as CodeModelInfo["PrismaModel"], userId),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<StatsCodeModelInfo["PrismaSelect"]>([["code", "Code"]], ...rest),
        visibility: {
            private: { code: ModelMap.get<CodeModelLogic>("Code").validate().visibility.private },
            public: { code: ModelMap.get<CodeModelLogic>("Code").validate().visibility.public },
            owner: (userId) => ({ code: ModelMap.get<CodeModelLogic>("Code").validate().visibility.owner(userId) }),
        },
    }),
});
