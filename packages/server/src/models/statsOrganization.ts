import { Prisma } from "@prisma/client";
import { StatsOrganization, StatsOrganizationSearchInput, StatsOrganizationSortBy } from "@shared/consts";
import i18next from "i18next";
import { selPad } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { defaultPermissions, oneIsPublic } from "../utils";
import { OrganizationModel } from "./organization";
import { ModelLogic } from "./types";

const __typename = 'StatsOrganization' as const;
const suppFields = [] as const;
export const StatsOrganizationModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: undefined,
    GqlUpdate: undefined,
    GqlModel: StatsOrganization,
    GqlSearch: StatsOrganizationSearchInput,
    GqlSort: StatsOrganizationSortBy,
    GqlPermission: {},
    PrismaCreate: Prisma.stats_organizationUpsertArgs['create'],
    PrismaUpdate: Prisma.stats_organizationUpsertArgs['update'],
    PrismaModel: Prisma.stats_organizationGetPayload<SelectWrap<Prisma.stats_organizationSelect>>,
    PrismaSelect: Prisma.stats_organizationSelect,
    PrismaWhere: Prisma.stats_organizationWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.stats_organization,
    display: {
        select: () => ({ id: true, organization: selPad(OrganizationModel.display.select) }),
        label: (select, languages) => i18next.t(`common:ObjectStats`, {
            lng: languages.length > 0 ? languages[0] : 'en',
            objectName: OrganizationModel.display.label(select.organization as any, languages),
        }),
    },
    format: {
        gqlRelMap: {
            __typename,
        },
        prismaRelMap: {
            __typename,
            organization: 'Organization',
        },
        countFields: {},
    },
    search: {
        defaultSort: StatsOrganizationSortBy.DateUpdatedDesc,
        sortBy: StatsOrganizationSortBy,
        searchFields: {
            periodTimeFrame: true,
            periodType: true,
        },
        searchStringQuery: () => ({ organization: OrganizationModel.search!.searchStringQuery() }),
    },
    validate: {
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            organization: 'Organization',
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => OrganizationModel.validate!.owner(data.organization as any),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.stats_organizationSelect>(data, [
            ['organization', 'Organization'],
        ], languages),
        visibility: {
            private: { organization: OrganizationModel.validate!.visibility.private },
            public: { organization: OrganizationModel.validate!.visibility.public },
            owner: (userId) => ({ organization: OrganizationModel.validate!.visibility.owner(userId) }),
        }
    },
})