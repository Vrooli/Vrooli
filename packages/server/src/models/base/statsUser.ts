import { DEFAULT_LANGUAGE, MaxObjects, StatsUserSortBy } from "@vrooli/shared";
import i18next from "i18next";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { oneIsPublic } from "../../utils/oneIsPublic.js";
import { defaultPermissions } from "../../validators/permissions.js";
import { StatsUserFormat } from "../formats.js";
import { ModelMap } from "./index.js";
import { type StatsUserModelInfo, type StatsUserModelLogic, type UserModelInfo, type UserModelLogic } from "./types.js";

const __typename = "StatsUser" as const;
export const StatsUserModel: StatsUserModelLogic = ({
    __typename,
    dbTable: "stats_user",
    display: () => ({
        label: {
            select: () => ({ id: true, user: { select: ModelMap.get<UserModelLogic>("User").display().label.select() } }),
            get: (select, languages) => i18next.t("common:ObjectStats", {
                lng: languages && languages.length > 0 ? languages[0] : DEFAULT_LANGUAGE,
                objectName: ModelMap.get<UserModelLogic>("User").display().label.get(select.user as UserModelInfo["DbModel"], languages),
            }),
        },
    }),
    format: StatsUserFormat,
    search: {
        defaultSort: StatsUserSortBy.PeriodStartAsc,
        sortBy: StatsUserSortBy,
        searchFields: {
            periodTimeFrame: true,
            periodType: true,
        },
        searchStringQuery: () => ({ user: ModelMap.get<UserModelLogic>("User").search.searchStringQuery() }),
    },
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            user: "User",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<UserModelLogic>("User").validate().owner(data?.user as UserModelInfo["DbModel"], userId),
        isDeleted: () => false,
        isPublic: (data, getParentInfo?) => oneIsPublic<StatsUserModelInfo["DbSelect"]>([["user", "User"]], data, getParentInfo),
        visibility: {
            own: function getOwn(data) {
                return {
                    user: useVisibility("User", "Own", data),
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    user: useVisibility("User", "OwnOrPublic", data),
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    user: useVisibility("User", "OwnPrivate", data),
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    user: useVisibility("User", "OwnPublic", data),
                };
            },
            public: function getPublic(data) {
                return {
                    user: useVisibility("User", "Public", data),
                };
            },
        },
    }),
});
