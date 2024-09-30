import { StatsApiSortBy } from "@local/shared";
import i18next from "i18next";
import { ModelMap } from ".";
import { useVisibility } from "../../builders/visibilityBuilder";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { StatsApiFormat } from "../formats";
import { ApiModelInfo, ApiModelLogic, StatsApiModelInfo, StatsApiModelLogic } from "./types";

const __typename = "StatsApi" as const;
export const StatsApiModel: StatsApiModelLogic = ({
    __typename,
    dbTable: "stats_api",
    display: () => ({
        label: {
            select: () => ({ id: true, api: { select: ModelMap.get<ApiModelLogic>("Api").display().label.select() } }),
            get: (select, languages) => i18next.t("common:ObjectStats", {
                lng: languages.length > 0 ? languages[0] : "en",
                objectName: ModelMap.get<ApiModelLogic>("Api").display().label.get(select.api as ApiModelInfo["PrismaModel"], languages),
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
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            api: "Api",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<ApiModelLogic>("Api").validate().owner(data?.api as ApiModelInfo["PrismaModel"], userId),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<StatsApiModelInfo["PrismaSelect"]>([["api", "Api"]], ...rest),
        visibility: {
            private: function getVisibilityPrivate(...params) {
                return {
                    api: useVisibility("Api", "private", ...params),
                };
            },
            public: function getVisibilityPublic(...params) {
                return {
                    api: useVisibility("Api", "public", ...params),
                };
            },
            owner: (...params) => ({ api: useVisibility("Api", "owner", ...params) }),
        },
    }),
});
