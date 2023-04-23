import { StatsRoutineSortBy } from "@local/consts";
import i18next from "i18next";
import { selPad } from "../builders";
import { defaultPermissions, oneIsPublic } from "../utils";
import { RoutineModel } from "./routine";
const __typename = "StatsRoutine";
const suppFields = [];
export const StatsRoutineModel = ({
    __typename,
    delegate: (prisma) => prisma.stats_routine,
    display: {
        select: () => ({ id: true, routine: selPad(RoutineModel.display.select) }),
        label: (select, languages) => i18next.t("common:ObjectStats", {
            lng: languages.length > 0 ? languages[0] : "en",
            objectName: RoutineModel.display.label(select.routine, languages),
        }),
    },
    format: {
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
        searchStringQuery: () => ({ routine: RoutineModel.search.searchStringQuery() }),
    },
    validate: {
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            routine: "Routine",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => RoutineModel.validate.owner(data.routine, userId),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic(data, [
            ["routine", "Routine"],
        ], languages),
        visibility: {
            private: { routine: RoutineModel.validate.visibility.private },
            public: { routine: RoutineModel.validate.visibility.public },
            owner: (userId) => ({ routine: RoutineModel.validate.visibility.owner(userId) }),
        },
    },
});
//# sourceMappingURL=statsRoutine.js.map