import { StatsUserSortBy } from "@local/shared";
import i18next from "i18next";
import { ModelMap } from ".";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { StatsUserFormat } from "../formats";
import { StatsUserModelInfo, StatsUserModelLogic, UserModelInfo, UserModelLogic } from "./types";

const __typename = "StatsUser" as const;
export const StatsUserModel: StatsUserModelLogic = ({
    __typename,
    dbTable: "stats_user",
    display: () => ({
        label: {
            select: () => ({ id: true, user: { select: ModelMap.get<UserModelLogic>("User").display().label.select() } }),
            get: (select, languages) => i18next.t("common:ObjectStats", {
                lng: languages.length > 0 ? languages[0] : "en",
                objectName: ModelMap.get<UserModelLogic>("User").display().label.get(select.user as UserModelInfo["PrismaModel"], languages),
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
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            user: "User",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<UserModelLogic>("User").validate().owner(data?.user as UserModelInfo["PrismaModel"], userId),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<StatsUserModelInfo["PrismaSelect"]>([["user", "User"]], ...rest),
        visibility: {
            private: { user: ModelMap.get<UserModelLogic>("User").validate().visibility.private },
            public: { user: ModelMap.get<UserModelLogic>("User").validate().visibility.public },
            owner: (userId) => ({ user: ModelMap.get<UserModelLogic>("User").validate().visibility.owner(userId) }),
        },
    }),
});
