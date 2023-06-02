import { StatsOrganizationSortBy } from "@local/shared";
import { Prisma } from "@prisma/client";
import i18next from "i18next";
import { selPad } from "../../builders";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { StatsOrganizationFormat } from "../format/statsOrganization";
import { ModelLogic } from "../types";
import { OrganizationModel } from "./organization";
import { StatsOrganizationModelLogic } from "./types";

const __typename = "StatsOrganization" as const;
const suppFields = [] as const;
export const StatsOrganizationModel: ModelLogic<StatsOrganizationModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.stats_organization,
    display: {
        label: {
            select: () => ({ id: true, organization: selPad(OrganizationModel.display.label.select) }),
            get: (select, languages) => i18next.t("common:ObjectStats", {
                lng: languages.length > 0 ? languages[0] : "en",
                objectName: OrganizationModel.display.label.get(select.organization as any, languages),
            }),
        },
    },
    format: StatsOrganizationFormat,
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
});
