import { MaxObjects, StatsRoutineSortBy } from "@local/shared";
import i18next from "i18next";
import { ModelMap } from ".";
import { useVisibility } from "../../builders/visibilityBuilder";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { StatsRoutineFormat } from "../formats";
import { RoutineModelInfo, RoutineModelLogic, StatsRoutineModelInfo, StatsRoutineModelLogic } from "./types";

const __typename = "StatsRoutine" as const;
export const StatsRoutineModel: StatsRoutineModelLogic = ({
    __typename,
    dbTable: "stats_routine",
    display: () => ({
        label: {
            select: () => ({ id: true, routine: { select: ModelMap.get<RoutineModelLogic>("Routine").display().label.select() } }),
            get: (select, languages) => i18next.t("common:ObjectStats", {
                lng: languages.length > 0 ? languages[0] : "en",
                objectName: ModelMap.get<RoutineModelLogic>("Routine").display().label.get(select.routine as RoutineModelInfo["PrismaModel"], languages),
            }),
        },
    }),
    format: StatsRoutineFormat,
    search: {
        defaultSort: StatsRoutineSortBy.PeriodStartAsc,
        sortBy: StatsRoutineSortBy,
        searchFields: {
            periodTimeFrame: true,
            periodType: true,
        },
        searchStringQuery: () => ({ routine: ModelMap.get<RoutineModelLogic>("Routine").search.searchStringQuery() }),
    },
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            routine: "Routine",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<RoutineModelLogic>("Routine").validate().owner(data?.routine as RoutineModelInfo["PrismaModel"], userId),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<StatsRoutineModelInfo["PrismaSelect"]>([["routine", "Routine"]], ...rest),
        visibility: {
            own: function getOwn(data) {
                return {
                    routine: useVisibility("Routine", "Own", data),
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    routine: useVisibility("Routine", "OwnOrPublic", data),
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    routine: useVisibility("Routine", "OwnPrivate", data),
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    routine: useVisibility("Routine", "OwnPublic", data),
                };
            },
            public: function getPublic(data) {
                return {
                    routine: useVisibility("Routine", "Public", data),
                };
            },
        },
    }),
});
