import { Prisma } from "@prisma/client";
import { StatsApi, StatsApiSearchInput, StatsApiSortBy } from "@shared/consts";
import i18next from "i18next";
import { selPad } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { defaultPermissions, oneIsPublic } from "../utils";
import { ApiModel } from "./api";
import { ModelLogic } from "./types";

const __typename = 'StatsApi' as const;
const suppFields = [] as const;
export const StatsApiModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: undefined,
    GqlUpdate: undefined,
    GqlModel: StatsApi,
    GqlSearch: StatsApiSearchInput,
    GqlSort: StatsApiSortBy,
    GqlPermission: {},
    PrismaCreate: Prisma.stats_apiUpsertArgs['create'],
    PrismaUpdate: Prisma.stats_apiUpsertArgs['update'],
    PrismaModel: Prisma.stats_apiGetPayload<SelectWrap<Prisma.stats_apiSelect>>,
    PrismaSelect: Prisma.stats_apiSelect,
    PrismaWhere: Prisma.stats_apiWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.stats_api,
    display: {
        select: () => ({ id: true, api: selPad(ApiModel.display.select) }),
        label: (select, languages) => (i18next as any).t(`common:ObjectStats`, {
            lng: languages.length > 0 ? languages[0] : 'en',
            objectName: ApiModel.display.label(select.api as any, languages),
        }),
    },
    format: {
        gqlRelMap: {
            __typename,
        },
        prismaRelMap: {
            __typename,
            api: 'Api',
        },
        countFields: {},
    },
    search: {
        defaultSort: StatsApiSortBy.DateUpdatedDesc,
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
            api: 'Api',
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => ApiModel.validate!.owner(data.api as any),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.stats_apiSelect>(data, [
            ['api', 'Api'],
        ], languages),
        visibility: {
            private: { api: ApiModel.validate!.visibility.private },
            public: { api: ApiModel.validate!.visibility.public },
            owner: (userId) => ({ api: ApiModel.validate!.visibility.owner(userId) }),
        }
    },
})