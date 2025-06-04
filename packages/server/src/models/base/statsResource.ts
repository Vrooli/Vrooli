import { DEFAULT_LANGUAGE, MaxObjects, StatsResourceSortBy } from "@local/shared";
import i18next from "i18next";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { oneIsPublic } from "../../utils/oneIsPublic.js";
import { StatsResourceFormat } from "../formats.js";
import { ModelMap } from "./index.js";
import { type ResourceModelInfo, type ResourceModelLogic, type StatsResourceModelInfo, type StatsResourceModelLogic } from "./types.js";

const __typename = "StatsResource" as const;
export const StatsResourceModel: StatsResourceModelLogic = ({
    __typename,
    dbTable: "stats_resource",
    display: () => ({
        label: {
            select: () => ({ id: true, resource: { select: ModelMap.get<ResourceModelLogic>("Resource").display().label.select() } }),
            get: (select, languages) => i18next.t("common:ObjectStats", {
                lng: languages && languages.length > 0 ? languages[0] : DEFAULT_LANGUAGE,
                objectName: ModelMap.get<ResourceModelLogic>("Resource").display().label.get(select.resource as ResourceModelInfo["DbModel"], languages),
            }),
        },
    }),
    format: StatsResourceFormat,
    search: {
        defaultSort: StatsResourceSortBy.PeriodStartAsc,
        sortBy: StatsResourceSortBy,
        searchFields: {
            periodTimeFrame: true,
            periodType: true,
        },
        searchStringQuery: () => ({ resource: ModelMap.get<ResourceModelLogic>("Resource").search.searchStringQuery() }),
    },
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            resource: "Resource",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<ResourceModelLogic>("Resource").validate().owner(data?.resource as ResourceModelInfo["DbModel"], userId),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<StatsResourceModelInfo["DbSelect"]>([["resource", "Resource"]], ...rest),
        visibility: {
            own: function getOwn(data) {
                return {
                    resource: useVisibility("Resource", "Own", data),
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    resource: useVisibility("Resource", "OwnOrPublic", data),
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    resource: useVisibility("Resource", "OwnPrivate", data),
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    resource: useVisibility("Resource", "OwnPublic", data),
                };
            },
            public: function getPublic(data) {
                return {
                    resource: useVisibility("Resource", "Public", data),
                };
            },
        },
    }),
});
