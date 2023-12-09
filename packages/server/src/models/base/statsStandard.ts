import { StatsStandardSortBy } from "@local/shared";
import i18next from "i18next";
import { ModelMap } from ".";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { StatsStandardFormat } from "../formats";
import { StandardModelInfo, StandardModelLogic, StatsStandardModelInfo, StatsStandardModelLogic } from "./types";

const __typename = "StatsStandard" as const;
export const StatsStandardModel: StatsStandardModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.stats_standard,
    display: () => ({
        label: {
            select: () => ({ id: true, standard: { select: ModelMap.get<StandardModelLogic>("Standard").display().label.select() } }),
            get: (select, languages) => i18next.t("common:ObjectStats", {
                lng: languages.length > 0 ? languages[0] : "en",
                objectName: ModelMap.get<StandardModelLogic>("Standard").display().label.get(select.standard as StandardModelInfo["PrismaModel"], languages),
            }),
        },
    }),
    format: StatsStandardFormat,
    search: {
        defaultSort: StatsStandardSortBy.PeriodStartAsc,
        sortBy: StatsStandardSortBy,
        searchFields: {
            periodTimeFrame: true,
            periodType: true,
        },
        searchStringQuery: () => ({ standard: ModelMap.get<StandardModelLogic>("Standard").search.searchStringQuery() }),
    },
    validate: () => ({
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            standard: "Standard",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<StandardModelLogic>("Standard").validate().owner(data?.standard as StandardModelInfo["PrismaModel"], userId),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<StatsStandardModelInfo["PrismaSelect"]>([["standard", "Standard"]], ...rest),
        visibility: {
            private: { standard: ModelMap.get<StandardModelLogic>("Standard").validate().visibility.private },
            public: { standard: ModelMap.get<StandardModelLogic>("Standard").validate().visibility.public },
            owner: (userId) => ({ standard: ModelMap.get<StandardModelLogic>("Standard").validate().visibility.owner(userId) }),
        },
    }),
});
