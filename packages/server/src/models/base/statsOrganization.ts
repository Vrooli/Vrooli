import { StatsOrganizationSortBy } from "@local/shared";
import i18next from "i18next";
import { ModelMap } from ".";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { StatsOrganizationFormat } from "../formats";
import { OrganizationModelInfo, OrganizationModelLogic, StatsOrganizationModelInfo, StatsOrganizationModelLogic } from "./types";

const __typename = "StatsOrganization" as const;
export const StatsOrganizationModel: StatsOrganizationModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.stats_organization,
    display: {
        label: {
            select: () => ({ id: true, organization: { select: ModelMap.get<OrganizationModelLogic>("Organization").display.label.select() } }),
            get: (select, languages) => i18next.t("common:ObjectStats", {
                lng: languages.length > 0 ? languages[0] : "en",
                objectName: ModelMap.get<OrganizationModelLogic>("Organization").display.label.get(select.organization as OrganizationModelInfo["PrismaModel"], languages),
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
        searchStringQuery: () => ({ organization: ModelMap.get<OrganizationModelLogic>("Organization").search.searchStringQuery() }),
    },
    validate: {
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            organization: "Organization",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<OrganizationModelLogic>("Organization").validate.owner(data?.organization as OrganizationModelInfo["PrismaModel"], userId),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<StatsOrganizationModelInfo["PrismaSelect"]>([["organization", "Organization"]], ...rest),
        visibility: {
            private: { organization: ModelMap.get<OrganizationModelLogic>("Organization").validate.visibility.private },
            public: { organization: ModelMap.get<OrganizationModelLogic>("Organization").validate.visibility.public },
            owner: (userId) => ({ organization: ModelMap.get<OrganizationModelLogic>("Organization").validate.visibility.owner(userId) }),
        },
    },
});
