import { MaxObjects, RunRoutineIOSortBy, runRoutineIOValidation } from "@local/shared";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { oneIsPublic } from "../../utils/oneIsPublic.js";
import { RunRoutineIOFormat } from "../formats.js";
import { ModelMap } from "./index.js";
import { RoutineVersionInputModelInfo, RoutineVersionInputModelLogic, RoutineVersionOutputModelInfo, RoutineVersionOutputModelLogic, RunRoutineIOModelInfo, RunRoutineIOModelLogic, RunRoutineModelInfo, RunRoutineModelLogic } from "./types.js";

const __typename = "RunRoutineIO" as const;
export const RunRoutineIOModel: RunRoutineIOModelLogic = ({
    __typename,
    dbTable: "run_routine_io",
    display: () => ({
        label: {
            select: () => ({
                id: true,
                runRoutine: { select: ModelMap.get<RunRoutineModelLogic>("RunRoutine").display().label.select() },
                routineVersionInput: { select: ModelMap.get<RoutineVersionInputModelLogic>("RoutineVersionInput").display().label.select() },
                routineVersionOutput: { select: ModelMap.get<RoutineVersionOutputModelLogic>("RoutineVersionOutput").display().label.select() },
            }),
            get: (select, languages) => {
                const runRoutineLabel = ModelMap.get<RunRoutineModelLogic>("RunRoutine").display().label.get(select.runRoutine as RunRoutineModelInfo["DbModel"], languages);
                const ioLabel = select.routineVersionInput
                    ? ModelMap.get<RoutineVersionInputModelLogic>("RoutineVersionInput").display().label.get(select.routineVersionInput as RoutineVersionInputModelInfo["DbModel"], languages)
                    : select.routineVersionOutput
                        ? ModelMap.get<RoutineVersionOutputModelLogic>("RoutineVersionOutput").display().label.get(select.routineVersionOutput as RoutineVersionOutputModelInfo["DbModel"], languages)
                        : "";
                if (runRoutineLabel.length > 0) {
                    return `${runRoutineLabel} - ${ioLabel}`;
                }
                return ioLabel;
            },
        },
    }),
    format: RunRoutineIOFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => {
                return {
                    id: data.id,
                    data: data.data,
                    nodeInputName: data.nodeInputName,
                    nodeName: data.nodeName,
                    runRoutine: await shapeHelper({ relation: "runRoutine", relTypes: ["Connect"], isOneToOne: true, objectType: "RunRoutine", parentRelationshipName: "io", data, ...rest }),
                    routineVersionInput: await shapeHelper({ relation: "routineVersionInput", relTypes: ["Connect"], isOneToOne: true, objectType: "RoutineVersionInput", parentRelationshipName: "runIO", data, ...rest }),
                    routineVersionOutput: await shapeHelper({ relation: "routineVersionOutput", relTypes: ["Connect"], isOneToOne: true, objectType: "RoutineVersionOutput", parentRelationshipName: "runIO", data, ...rest }),
                };
            },
            update: async ({ data }) => {
                return {
                    data: data.data,
                };
            },
        },
        yup: runRoutineIOValidation,
    },
    search: {
        defaultSort: RunRoutineIOSortBy.DateUpdatedDesc,
        sortBy: RunRoutineIOSortBy,
        searchFields: {
            createdTimeFrame: true,
            excludeIds: true,
            runRoutineIds: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({ runRoutine: ModelMap.get<RunRoutineModelLogic>("RunRoutine").search.searchStringQuery() }),
    },
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            runRoutine: "RunRoutine",
        }),
        permissionResolvers: defaultPermissions,
        profanityFields: ["data"],
        owner: (data, userId) => ModelMap.get<RunRoutineModelLogic>("RunRoutine").validate().owner(data?.runRoutine as RunRoutineModelInfo["DbModel"], userId),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<RunRoutineIOModelInfo["DbSelect"]>([["runRoutine", "RunRoutine"]], ...rest),
        visibility: {
            own: function getOwn(data) {
                return {
                    runRoutine: useVisibility("RunRoutine", "Own", data),
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    runRoutine: useVisibility("RunRoutine", "OwnOrPublic", data),
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    runRoutine: useVisibility("RunRoutine", "OwnPrivate", data),
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    runRoutine: useVisibility("RunRoutine", "OwnPublic", data),
                };
            },
            public: function getPublic(data) {
                return {
                    runRoutine: useVisibility("RunRoutine", "Public", data),
                };
            },
        },
    }),
});
