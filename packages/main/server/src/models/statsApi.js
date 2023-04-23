import { StatsApiSortBy } from "@local/consts";
import i18next from "i18next";
import { selPad } from "../builders";
import { defaultPermissions, oneIsPublic } from "../utils";
import { ApiModel } from "./api";
const __typename = "StatsApi";
const suppFields = [];
export const StatsApiModel = ({
    __typename,
    delegate: (prisma) => prisma.stats_api,
    display: {
        select: () => ({ id: true, api: selPad(ApiModel.display.select) }),
        label: (select, languages) => i18next.t("common:ObjectStats", {
            lng: languages.length > 0 ? languages[0] : "en",
            objectName: ApiModel.display.label(select.api, languages),
        }),
    },
    format: {
        gqlRelMap: {
            __typename,
        },
        prismaRelMap: {
            __typename,
            api: "Api",
        },
        countFields: {},
    },
    search: {
        defaultSort: StatsApiSortBy.PeriodStartAsc,
        sortBy: StatsApiSortBy,
        searchFields: {
            periodTimeFrame: true,
            periodType: true,
        },
        searchStringQuery: () => ({ api: ApiModel.search.searchStringQuery() }),
    },
    validate: {
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            api: "Api",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ApiModel.validate.owner(data.api, userId),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic(data, [
            ["api", "Api"],
        ], languages),
        visibility: {
            private: { api: ApiModel.validate.visibility.private },
            public: { api: ApiModel.validate.visibility.public },
            owner: (userId) => ({ api: ApiModel.validate.visibility.owner(userId) }),
        },
    },
});
//# sourceMappingURL=statsApi.js.map