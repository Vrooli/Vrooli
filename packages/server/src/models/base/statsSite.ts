import { StatsSiteSortBy } from "@local/shared";
import i18next from "i18next";
import { defaultPermissions } from "../../utils";
import { StatsSiteFormat } from "../format/statsSite";
import { ModelLogic } from "../types";
import { StatsSiteModelLogic } from "./types";

const __typename = "StatsSite" as const;
const suppFields = [] as const;
export const StatsSiteModel: ModelLogic<StatsSiteModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.stats_site,
    display: {
        label: {
            select: () => ({ id: true }),
            get: (_, languages) => i18next.t("common:SiteStats", {
                lng: languages.length > 0 ? languages[0] : "en",
            }),
        },
    },
    format: StatsSiteFormat,
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
        permissionResolvers: (params) => defaultPermissions({ ...params, isAdmin: false }), // Force isAdmin false, since there is no "visibility.owner" query
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
