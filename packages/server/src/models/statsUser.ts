import { StatsUser, StatsUserSearchInput, StatsUserSortBy } from "@local/shared";
import { Prisma } from "@prisma/client";
import i18next from "i18next";
import { selPad } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { defaultPermissions, oneIsPublic } from "../utils";
import { ModelLogic } from "./types";
import { UserModel } from "./user";

const __typename = "StatsUser" as const;
const suppFields = [] as const;
export const StatsUserModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: undefined,
    GqlUpdate: undefined,
    GqlModel: StatsUser,
    GqlSearch: StatsUserSearchInput,
    GqlSort: StatsUserSortBy,
    GqlPermission: {},
    PrismaCreate: Prisma.stats_userUpsertArgs["create"],
    PrismaUpdate: Prisma.stats_userUpsertArgs["update"],
    PrismaModel: Prisma.stats_userGetPayload<SelectWrap<Prisma.stats_userSelect>>,
    PrismaSelect: Prisma.stats_userSelect,
    PrismaWhere: Prisma.stats_userWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.stats_user,
    display: {
        select: () => ({ id: true, user: selPad(UserModel.display.select) }),
        label: (select, languages) => i18next.t("common:ObjectStats", {
            lng: languages.length > 0 ? languages[0] : "en",
            objectName: UserModel.display.label(select.user as any, languages),
        }),
    },
    format: {
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
});
