import { StatsStandard, StatsStandardSearchInput, StatsStandardSortBy } from "@local/shared";
import { Prisma } from "@prisma/client";
import i18next from "i18next";
import { selPad } from "../../builders";
import { SelectWrap } from "../../builders/types";
import { PrismaType } from "../../types";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { StandardModel } from "./standard";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "StatsStandard" as const;
export const StatsStandardFormat: Formatter<ModelStatsStandardLogic> = {
        gqlRelMap: {
            __typename,
        },
        prismaRelMap: {
            __typename,
            standard: "Standard",
        },
        countFields: {},
    },
    search: {
        defaultSort: StatsStandardSortBy.PeriodStartAsc,
        sortBy: StatsStandardSortBy,
        searchFields: {
            periodTimeFrame: true,
            periodType: true,
        },
        searchStringQuery: () => ({ standard: StandardModel.search!.searchStringQuery() }),
    },
    validate: {
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            standard: "Standard",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => StandardModel.validate!.owner(data.standard as any, userId),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.stats_standardSelect>(data, [
            ["standard", "Standard"],
        ], languages),
        visibility: {
            private: { standard: StandardModel.validate!.visibility.private },
            public: { standard: StandardModel.validate!.visibility.public },
            owner: (userId) => ({ standard: StandardModel.validate!.visibility.owner(userId) }),
        },
    },
};
