import { StatsSiteSortBy } from "@local/consts";
import i18next from "i18next";
import { defaultPermissions } from "../utils";
const __typename = "StatsSite";
const suppFields = [];
export const StatsSiteModel = ({
    __typename,
    delegate: (prisma) => prisma.stats_site,
    display: {
        select: () => ({ id: true }),
        label: (_, languages) => i18next.t("common:SiteStats", {
            lng: languages.length > 0 ? languages[0] : "en",
        }),
    },
    format: {
        gqlRelMap: {
            __typename,
        },
        prismaRelMap: {
            __typename,
        },
        countFields: {},
    },
    search: {
        defaultSort: StatsSiteSortBy.PeriodStartAsc,
        sortBy: StatsSiteSortBy,
        searchFields: {
            periodTimeFrame: true,
            periodType: true,
        },
        searchStringQuery: () => ({}),
    },
    validate: {
        isDeleted: () => false,
        isPublic: () => true,
        isTransferable: false,
        maxObjects: 10000000,
        owner: () => ({}),
        permissionResolvers: (params) => defaultPermissions({ ...params, isAdmin: false }),
        permissionsSelect: () => ({
            id: true,
        }),
        visibility: {
            private: {},
            public: {},
            owner: () => ({}),
        },
    },
});
//# sourceMappingURL=statsSite.js.map