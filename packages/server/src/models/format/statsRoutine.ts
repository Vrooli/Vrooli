import { StatsRoutine, StatsRoutineSearchInput, StatsRoutineSortBy } from "@local/shared";
import { Prisma } from "@prisma/client";
import i18next from "i18next";
import { selPad } from "../../builders";
import { SelectWrap } from "../../builders/types";
import { PrismaType } from "../../types";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { RoutineModel } from "./routine";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "StatsRoutine" as const;
export const StatsRoutineFormat: Formatter<ModelStatsRoutineLogic> = {
        gqlRelMap: {
            __typename,
        },
        prismaRelMap: {
            __typename,
            routine: "Routine",
        },
        countFields: {},
    },
    search: {
        defaultSort: StatsRoutineSortBy.PeriodStartAsc,
        sortBy: StatsRoutineSortBy,
        searchFields: {
            periodTimeFrame: true,
            periodType: true,
        },
        searchStringQuery: () => ({ routine: RoutineModel.search!.searchStringQuery() }),
    },
    validate: {
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            routine: "Routine",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => RoutineModel.validate!.owner(data.routine as any, userId),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.stats_routineSelect>(data, [
            ["routine", "Routine"],
        ], languages),
        visibility: {
            private: { routine: RoutineModel.validate!.visibility.private },
            public: { routine: RoutineModel.validate!.visibility.public },
            owner: (userId) => ({ routine: RoutineModel.validate!.visibility.owner(userId) }),
        },
    },
};
