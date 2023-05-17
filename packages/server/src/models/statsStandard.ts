import { StatsStandard, StatsStandardSearchInput, StatsStandardSortBy } from "@local/shared";
import { Prisma } from "@prisma/client";
import i18next from "i18next";
import { selPad } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { defaultPermissions, oneIsPublic } from "../utils";
import { StandardModel } from "./standard";
import { ModelLogic } from "./types";

const __typename = "StatsStandard" as const;
const suppFields = [] as const;
export const StatsStandardModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: undefined,
    GqlUpdate: undefined,
    GqlModel: StatsStandard,
    GqlSearch: StatsStandardSearchInput,
    GqlSort: StatsStandardSortBy,
    GqlPermission: object,
    PrismaCreate: Prisma.stats_standardUpsertArgs["create"],
    PrismaUpdate: Prisma.stats_standardUpsertArgs["update"],
    PrismaModel: Prisma.stats_standardGetPayload<SelectWrap<Prisma.stats_standardSelect>>,
    PrismaSelect: Prisma.stats_standardSelect,
    PrismaWhere: Prisma.stats_standardWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.stats_standard,
    display: {
        label: {
            select: () => ({ id: true, standard: selPad(StandardModel.display.label.select) }),
            get: (select, languages) => i18next.t("common:ObjectStats", {
                lng: languages.length > 0 ? languages[0] : "en",
                objectName: StandardModel.display.label.get(select.standard as any, languages),
            }),
        },
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
        owner: (data, userId) => StandardModel.validate!.owner(data.standard as any, userId),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.stats_standardSelect>(data, [
            ["standard", "Standard"],
        ], languages),
        visibility: {
            private: { standard: StandardModel.validate!.visibility.private },
            public: { standard: StandardModel.validate!.visibility.public },
            owner: (userId) => ({ standard: StandardModel.validate!.visibility.owner(userId) }),
        },
    },
});
