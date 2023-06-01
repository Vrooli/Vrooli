import { StatsSite, StatsSiteSearchInput, StatsSiteSortBy } from "@local/shared";
import { Prisma } from "@prisma/client";
import i18next from "i18next";
import { SelectWrap } from "../../builders/types";
import { PrismaType } from "../../types";
import { defaultPermissions } from "../../utils";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "StatsSite" as const;
export const StatsSiteFormat: Formatter<ModelStatsSiteLogic> = {
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
};
