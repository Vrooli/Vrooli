import { StatsStandardSortBy } from "@local/consts";
import i18next from "i18next";
import { selPad } from "../builders";
import { defaultPermissions, oneIsPublic } from "../utils";
import { StandardModel } from "./standard";
const __typename = "StatsStandard";
const suppFields = [];
export const StatsStandardModel = ({
    __typename,
    delegate: (prisma) => prisma.stats_standard,
    display: {
        select: () => ({ id: true, standard: selPad(StandardModel.display.select) }),
        label: (select, languages) => i18next.t("common:ObjectStats", {
            lng: languages.length > 0 ? languages[0] : "en",
            objectName: StandardModel.display.label(select.standard, languages),
        }),
    },
    format: {
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
        searchStringQuery: () => ({ standard: StandardModel.search.searchStringQuery() }),
    },
    validate: {
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            standard: "Standard",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => StandardModel.validate.owner(data.standard, userId),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic(data, [
            ["standard", "Standard"],
        ], languages),
        visibility: {
            private: { standard: StandardModel.validate.visibility.private },
            public: { standard: StandardModel.validate.visibility.public },
            owner: (userId) => ({ standard: StandardModel.validate.visibility.owner(userId) }),
        },
    },
});
//# sourceMappingURL=statsStandard.js.map