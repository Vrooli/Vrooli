import { DEFAULT_LANGUAGE, MaxObjects, StatsSiteSortBy } from "@vrooli/shared";
import i18next from "i18next";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { StatsSiteFormat } from "../formats.js";
import { type StatsSiteModelLogic } from "./types.js";

const __typename = "StatsSite" as const;
export const StatsSiteModel: StatsSiteModelLogic = ({
    __typename,
    dbTable: "stats_site",
    display: () => ({
        label: {
            select: () => ({ id: true }),
            get: (_, languages) => i18next.t("common:SiteStats", {
                lng: languages && languages.length > 0 ? languages[0] : DEFAULT_LANGUAGE,
            }),
        },
    }),
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
    validate: () => ({
        isDeleted: () => false,
        isPublic: () => true,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: () => ({}),
        permissionResolvers: (params) => defaultPermissions({ ...params, isAdmin: false }), // Force isAdmin false, since there is no "visibility.owner" query
        permissionsSelect: () => ({
            id: true,
        }),
        visibility: {
            own: null, // Search method disabled, since no one owns site stats
            // Always public, os it's the same as "public"
            ownOrPublic: function getOwnOrPublic(data) {
                return useVisibility("StatsSite", "Public", data);
            },
            ownPrivate: null, // Search method disabled, since no one owns site stats
            ownPublic: null, // Search method disabled, since no one owns site stats
            public: function getVisibilityPublic() {
                return {}; // Intentionally empty, since site stats are always public
            },
        },
    }),
});
