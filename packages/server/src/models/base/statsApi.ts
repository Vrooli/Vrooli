import { DEFAULT_LANGUAGE, MaxObjects, StatsApiSortBy } from "@local/shared";
import i18next from "i18next";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { oneIsPublic } from "../../utils/oneIsPublic.js";
import { StatsApiFormat } from "../formats.js";
import { ModelMap } from "./index.js";
import { ApiModelInfo, ApiModelLogic, StatsApiModelInfo, StatsApiModelLogic } from "./types.js";

const __typename = "StatsApi" as const;
export const StatsApiModel: StatsApiModelLogic = ({
    __typename,
    dbTable: "stats_api",
    display: () => ({
        label: {
            select: () => ({ id: true, api: { select: ModelMap.get<ApiModelLogic>("Api").display().label.select() } }),
            get: (select, languages) => i18next.t("common:ObjectStats", {
                lng: languages && languages.length > 0 ? languages[0] : DEFAULT_LANGUAGE,
                objectName: ModelMap.get<ApiModelLogic>("Api").display().label.get(select.api as ApiModelInfo["DbModel"], languages),
            }),
        },
    }),
    format: StatsApiFormat,
    search: {
        defaultSort: StatsApiSortBy.PeriodStartAsc,
        sortBy: StatsApiSortBy,
        searchFields: {
            periodTimeFrame: true,
            periodType: true,
        },
        searchStringQuery: () => ({ api: ModelMap.get<ApiModelLogic>("Api").search.searchStringQuery() }),
    },
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            api: "Api",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<ApiModelLogic>("Api").validate().owner(data?.api as ApiModelInfo["DbModel"], userId),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<StatsApiModelInfo["DbSelect"]>([["api", "Api"]], ...rest),
        visibility: {
            own: function getOwn(data) {
                return {
                    api: useVisibility("Api", "Own", data),
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    api: useVisibility("Api", "OwnOrPublic", data),
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    api: useVisibility("Api", "OwnPrivate", data),
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    api: useVisibility("Api", "OwnPublic", data),
                };
            },
            public: function getPublic(data) {
                return {
                    api: useVisibility("Api", "Public", data),
                };
            },
        },
    }),
});
