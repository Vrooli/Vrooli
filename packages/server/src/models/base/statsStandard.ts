import { DEFAULT_LANGUAGE, MaxObjects, StatsStandardSortBy } from "@local/shared";
import i18next from "i18next";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { oneIsPublic } from "../../utils/oneIsPublic.js";
import { StatsStandardFormat } from "../formats.js";
import { ModelMap } from "./index.js";
import { StandardModelInfo, StandardModelLogic, StatsStandardModelInfo, StatsStandardModelLogic } from "./types.js";

const __typename = "StatsStandard" as const;
export const StatsStandardModel: StatsStandardModelLogic = ({
    __typename,
    dbTable: "stats_standard",
    display: () => ({
        label: {
            select: () => ({ id: true, standard: { select: ModelMap.get<StandardModelLogic>("Standard").display().label.select() } }),
            get: (select, languages) => i18next.t("common:ObjectStats", {
                lng: languages && languages.length > 0 ? languages[0] : DEFAULT_LANGUAGE,
                objectName: ModelMap.get<StandardModelLogic>("Standard").display().label.get(select.standard as StandardModelInfo["DbModel"], languages),
            }),
        },
    }),
    format: StatsStandardFormat,
    search: {
        defaultSort: StatsStandardSortBy.PeriodStartAsc,
        sortBy: StatsStandardSortBy,
        searchFields: {
            periodTimeFrame: true,
            periodType: true,
        },
        searchStringQuery: () => ({ standard: ModelMap.get<StandardModelLogic>("Standard").search.searchStringQuery() }),
    },
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            standard: "Standard",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<StandardModelLogic>("Standard").validate().owner(data?.standard as StandardModelInfo["DbModel"], userId),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<StatsStandardModelInfo["DbSelect"]>([["standard", "Standard"]], ...rest),
        visibility: {
            own: function getOwn(data) {
                return {
                    standard: useVisibility("Standard", "Own", data),
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    standard: useVisibility("Standard", "OwnOrPublic", data),
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    standard: useVisibility("Standard", "OwnPrivate", data),
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    standard: useVisibility("Standard", "OwnPublic", data),
                };
            },
            public: function getPublic(data) {
                return {
                    standard: useVisibility("Standard", "Public", data),
                };
            },
        },
    }),
});
