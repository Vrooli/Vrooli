import { StatsRoutineSortBy } from "@local/shared";
import { Prisma } from "@prisma/client";
import i18next from "i18next";
import { selPad } from "../../builders";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { StatsRoutineFormat } from "../format/statsRoutine";
import { ModelLogic } from "../types";
import { RoutineModel } from "./routine";
import { StatsRoutineModelLogic } from "./types";

const __typename = "StatsRoutine" as const;
const suppFields = [] as const;
export const StatsRoutineModel: ModelLogic<StatsRoutineModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.stats_routine,
    display: {
        label: {
            select: () => ({ id: true, routine: selPad(RoutineModel.display.label.select) }),
            get: (select, languages) => i18next.t("common:ObjectStats", {
                lng: languages.length > 0 ? languages[0] : "en",
                objectName: RoutineModel.display.label.get(select.routine as any, languages),
            }),
        },
    },
    format: StatsRoutineFormat,
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
});