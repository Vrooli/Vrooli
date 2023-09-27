import { StatsOrganizationSortBy } from "@local/shared";
import i18next from "i18next";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { StatsOrganizationFormat } from "../formats";
import { ModelLogic } from "../types";
import { OrganizationModel } from "./organization";
import { OrganizationModelLogic, StatsOrganizationModelLogic } from "./types";

const __typename = "StatsOrganization" as const;
const suppFields = [] as const;
export const StatsOrganizationModel: ModelLogic<StatsOrganizationModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.stats_organization,
    display: {
        label: {
            select: () => ({ id: true, organization: { select: OrganizationModel.display.label.select() } }),
            get: (select, languages) => i18next.t("common:ObjectStats", {
                lng: languages.length > 0 ? languages[0] : "en",
                objectName: OrganizationModel.display.label.get(select.organization as OrganizationModelLogic["PrismaModel"], languages),
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
        searchStringQuery: () => ({ organization: OrganizationModel.search.searchStringQuery() }),
    },
    validate: {
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            organization: "Organization",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => OrganizationModel.validate.owner(data?.organization as OrganizationModelLogic["PrismaModel"], userId),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<StatsOrganizationModelLogic["PrismaSelect"]>([["organization", "Organization"]], ...rest),
        visibility: {
            private: { organization: OrganizationModel.validate.visibility.private },
            public: { organization: OrganizationModel.validate.visibility.public },
            owner: (userId) => ({ organization: OrganizationModel.validate.visibility.owner(userId) }),
        },
    },
});
