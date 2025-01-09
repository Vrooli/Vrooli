import { DEFAULT_LANGUAGE, MaxObjects, StatsCodeSortBy } from "@local/shared";
import i18next from "i18next";
import { ModelMap } from ".";
import { useVisibility } from "../../builders/visibilityBuilder";
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
                lng: languages && languages.length > 0 ? languages[0] : DEFAULT_LANGUAGE,
                objectName: ModelMap.get<CodeModelLogic>("Code").display().label.get(select.code as CodeModelInfo["DbModel"], languages),
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
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            code: "Code",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<CodeModelLogic>("Code").validate().owner(data?.code as CodeModelInfo["DbModel"], userId),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<StatsCodeModelInfo["DbSelect"]>([["code", "Code"]], ...rest),
        visibility: {
            own: function getOwn(data) {
                return {
                    code: useVisibility("Code", "Own", data),
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    code: useVisibility("Code", "OwnOrPublic", data),
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    code: useVisibility("Code", "OwnPrivate", data),
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    code: useVisibility("Code", "OwnPublic", data),
                };
            },
            public: function getPublic(data) {
                return {
                    code: useVisibility("Code", "Public", data),
                };
            },
        },
    }),
});
