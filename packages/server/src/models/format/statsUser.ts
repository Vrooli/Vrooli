import { StatsUser, StatsUserSearchInput, StatsUserSortBy } from "@local/shared";
import { Prisma } from "@prisma/client";
import i18next from "i18next";
import { selPad } from "../../builders";
import { SelectWrap } from "../../builders/types";
import { PrismaType } from "../../types";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { ModelLogic } from "./types";
import { UserModel } from "./user";
import { Formatter } from "../types";

const __typename = "StatsUser" as const;
export const StatsUserFormat: Formatter<ModelStatsUserLogic> = {
        gqlRelMap: {
            __typename,
        },
        prismaRelMap: {
            __typename,
            user: "User",
        },
        countFields: {},
    },
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
};
