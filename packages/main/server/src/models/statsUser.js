import { StatsUserSortBy } from "@local/consts";
import i18next from "i18next";
import { selPad } from "../builders";
import { defaultPermissions, oneIsPublic } from "../utils";
import { UserModel } from "./user";
const __typename = "StatsUser";
const suppFields = [];
export const StatsUserModel = ({
    __typename,
    delegate: (prisma) => prisma.stats_user,
    display: {
        select: () => ({ id: true, user: selPad(UserModel.display.select) }),
        label: (select, languages) => i18next.t("common:ObjectStats", {
            lng: languages.length > 0 ? languages[0] : "en",
            objectName: UserModel.display.label(select.user, languages),
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
        searchStringQuery: () => ({ user: UserModel.search.searchStringQuery() }),
    },
    validate: {
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            user: "User",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => UserModel.validate.owner(data.user, userId),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic(data, [
            ["user", "User"],
        ], languages),
        visibility: {
            private: { user: UserModel.validate.visibility.private },
            public: { user: UserModel.validate.visibility.public },
            owner: (userId) => ({ user: UserModel.validate.visibility.owner(userId) }),
        },
    },
});
//# sourceMappingURL=statsUser.js.map