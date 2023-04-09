import { Prisma } from "@prisma/client";
import { StatsSite, StatsSiteSearchInput, StatsSiteSortBy } from "@shared/consts";
import i18next from "i18next";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { defaultPermissions } from "../utils";
import { ModelLogic } from "./types";

const __typename = 'StatsSite' as const;
const suppFields = [] as const;
export const StatsSiteModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: undefined,
    GqlUpdate: undefined,
    GqlModel: StatsSite,
    GqlSearch: StatsSiteSearchInput,
    GqlSort: StatsSiteSortBy,
    GqlPermission: {},
    PrismaCreate: Prisma.stats_siteUpsertArgs['create'],
    PrismaUpdate: Prisma.stats_siteUpsertArgs['update'],
    PrismaModel: Prisma.stats_siteGetPayload<SelectWrap<Prisma.stats_siteSelect>>,
    PrismaSelect: Prisma.stats_siteSelect,
    PrismaWhere: Prisma.stats_siteWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.stats_site,
    display: {
        select: () => ({ id: true }),
        label: (_, languages) => i18next.t(`common:SiteStats`, {
            lng: languages.length > 0 ? languages[0] : 'en',
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
        searchStringQuery: () => ({})
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
})