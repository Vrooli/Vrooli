import { StatsOrganization, StatsOrganizationSearchInput, StatsOrganizationSortBy } from "@local/shared";
import { Prisma } from "@prisma/client";
import i18next from "i18next";
import { selPad } from "../../builders";
import { SelectWrap } from "../../builders/types";
import { PrismaType } from "../../types";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { OrganizationModel } from "./organization";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "StatsOrganization" as const;
export const StatsOrganizationFormat: Formatter<ModelStatsOrganizationLogic> = {
        gqlRelMap: {
            __typename,
        },
        prismaRelMap: {
            __typename,
            organization: "Organization",
        },
        countFields: {},
    },
    search: {
        defaultSort: StatsOrganizationSortBy.PeriodStartAsc,
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
            organization: "Organization",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => OrganizationModel.validate!.owner(data.organization as any, userId),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.stats_organizationSelect>(data, [
            ["organization", "Organization"],
        ], languages),
        visibility: {
            private: { organization: OrganizationModel.validate!.visibility.private },
            public: { organization: OrganizationModel.validate!.visibility.public },
            owner: (userId) => ({ organization: OrganizationModel.validate!.visibility.owner(userId) }),
        },
    },
};
