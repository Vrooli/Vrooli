import { StatsApi, StatsApiSearchInput, StatsApiSortBy } from "@local/shared";
import { Prisma } from "@prisma/client";
import i18next from "i18next";
import { selPad } from "../../builders";
import { SelectWrap } from "../../builders/types";
import { PrismaType } from "../../types";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { ApiModel } from "./api";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "StatsApi" as const;
export const StatsApiFormat: Formatter<ModelStatsApiLogic> = {
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
        searchStringQuery: () => ({ api: ApiModel.search!.searchStringQuery() }),
    },
    validate: {
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            api: "Api",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ApiModel.validate!.owner(data.api as any, userId),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.stats_apiSelect>(data, [
            ["api", "Api"],
        ], languages),
        visibility: {
            private: { api: ApiModel.validate!.visibility.private },
            public: { api: ApiModel.validate!.visibility.public },
            owner: (userId) => ({ api: ApiModel.validate!.visibility.owner(userId) }),
        },
    },
};
