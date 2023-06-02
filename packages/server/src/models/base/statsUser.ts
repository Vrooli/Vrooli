import { StatsUserSortBy } from "@local/shared";
import { Prisma } from "@prisma/client";
import i18next from "i18next";
import { selPad } from "../../builders";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { StatsUserFormat } from "../format/statsUser";
import { ModelLogic } from "../types";
import { StatsUserModelLogic } from "./types";
import { UserModel } from "./user";

const __typename = "StatsUser" as const;
const suppFields = [] as const;
export const StatsUserModel: ModelLogic<StatsUserModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.stats_user,
    display: {
        label: {
            select: () => ({ id: true, user: selPad(UserModel.display.label.select) }),
            get: (select, languages) => i18next.t("common:ObjectStats", {
                lng: languages.length > 0 ? languages[0] : "en",
                objectName: UserModel.display.label.get(select.user as any, languages),
            }),
        },
    },
    format: StatsUserFormat,
    search: {
        defaultSort: StatsUserSortBy.PeriodStartAsc,
        sortBy: StatsUserSortBy,
        searchFields: {
            periodTimeFrame: true,
            periodType: true,
        },
        searchStringQuery: () => ({ user: UserModel.search!.searchStringQuery() }),
    },
    validate: {
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            user: "User",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => UserModel.validate!.owner(data.user as any, userId),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.stats_userSelect>(data, [
            ["user", "User"],
        ], languages),
        visibility: {
            private: { user: UserModel.validate!.visibility.private },
            public: { user: UserModel.validate!.visibility.public },
            owner: (userId) => ({ user: UserModel.validate!.visibility.owner(userId) }),
        },
    },
});
