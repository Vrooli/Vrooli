import { StatsOrganizationSortBy } from "@local/consts";
import i18next from "i18next";
import { selPad } from "../builders";
import { defaultPermissions, oneIsPublic } from "../utils";
import { OrganizationModel } from "./organization";
const __typename = "StatsOrganization";
const suppFields = [];
export const StatsOrganizationModel = ({
    __typename,
    delegate: (prisma) => prisma.stats_organization,
    display: {
        select: () => ({ id: true, organization: selPad(OrganizationModel.display.select) }),
        label: (select, languages) => i18next.t("common:ObjectStats", {
            lng: languages.length > 0 ? languages[0] : "en",
            objectName: OrganizationModel.display.label(select.organization, languages),
        }),
    },
    format: {
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
        owner: (data, userId) => OrganizationModel.validate.owner(data.organization, userId),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic(data, [
            ["organization", "Organization"],
        ], languages),
        visibility: {
            private: { organization: OrganizationModel.validate.visibility.private },
            public: { organization: OrganizationModel.validate.visibility.public },
            owner: (userId) => ({ organization: OrganizationModel.validate.visibility.owner(userId) }),
        },
    },
});
//# sourceMappingURL=statsOrganization.js.map