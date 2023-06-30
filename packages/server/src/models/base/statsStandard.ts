import { StatsStandardSortBy } from "@local/shared";
import { Prisma } from "@prisma/client";
import i18next from "i18next";
import { selPad } from "../../builders";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { StatsStandardFormat } from "../format/statsStandard";
import { ModelLogic } from "../types";
import { StandardModel } from "./standard";
import { StatsStandardModelLogic } from "./types";

const __typename = "StatsStandard" as const;
const suppFields = [] as const;
export const StatsStandardModel: ModelLogic<StatsStandardModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.stats_standard,
    display: {
        label: {
            select: () => ({ id: true, standard: selPad(StandardModel.display.label.select) }),
            get: (select, languages) => i18next.t("common:ObjectStats", {
                lng: languages.length > 0 ? languages[0] : "en",
                objectName: StandardModel.display.label.get(select.standard as any, languages),
            }),
        },
    },
    format: StatsStandardFormat,
    search: {
        defaultSort: StatsStandardSortBy.PeriodStartAsc,
        sortBy: StatsStandardSortBy,
        searchFields: {
            periodTimeFrame: true,
            periodType: true,
        },
        searchStringQuery: () => ({ standard: StandardModel.search!.searchStringQuery() }),
    },
    validate: {
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            standard: "Standard",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => StandardModel.validate.owner(data.standard as any, userId),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.stats_standardSelect>(data, [
            ["standard", "Standard"],
        ], languages),
        visibility: {
            private: { standard: StandardModel.validate.visibility.private },
            public: { standard: StandardModel.validate.visibility.public },
            owner: (userId) => ({ standard: StandardModel.validate.visibility.owner(userId) }),
        },
    },
});
