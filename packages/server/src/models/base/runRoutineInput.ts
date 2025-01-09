import { MaxObjects, RunRoutineInputSortBy, runRoutineInputValidation } from "@local/shared";
import { ModelMap } from ".";
import { shapeHelper } from "../../builders/shapeHelper";
import { useVisibility } from "../../builders/visibilityBuilder";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { RunRoutineInputFormat } from "../formats";
import { RoutineVersionInputModelInfo, RoutineVersionInputModelLogic, RunRoutineInputModelInfo, RunRoutineInputModelLogic, RunRoutineModelInfo, RunRoutineModelLogic } from "./types";

const __typename = "RunRoutineInput" as const;
export const RunRoutineInputModel: RunRoutineInputModelLogic = ({
    __typename,
    dbTable: "run_routine_input",
    display: () => ({
        label: {
            select: () => ({
                id: true,
                input: { select: ModelMap.get<RoutineVersionInputModelLogic>("RoutineVersionInput").display().label.select() },
                runRoutine: { select: ModelMap.get<RunRoutineModelLogic>("RunRoutine").display().label.select() },
            }),
            // Label combines runRoutine's label and input's label
            get: (select, languages) => {
                const runRoutineLabel = ModelMap.get<RunRoutineModelLogic>("RunRoutine").display().label.get(select.runRoutine as RunRoutineModelInfo["DbModel"], languages);
                const inputLabel = ModelMap.get<RoutineVersionInputModelLogic>("RoutineVersionInput").display().label.get(select.input as RoutineVersionInputModelInfo["DbModel"], languages);
                if (runRoutineLabel.length > 0) {
                    return `${runRoutineLabel} - ${inputLabel}`;
                }
                return inputLabel;
            },
        },
    }),
    format: RunRoutineInputFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => {
                return {
                    id: data.id,
                    data: data.data,
                    input: await shapeHelper({ relation: "input", relTypes: ["Connect"], isOneToOne: true, objectType: "RoutineVersionInput", parentRelationshipName: "runInputs", data, ...rest }),
                    runRoutine: await shapeHelper({ relation: "runRoutine", relTypes: ["Connect"], isOneToOne: true, objectType: "RunRoutine", parentRelationshipName: "inputs", data, ...rest }),
                };
            },
            update: async ({ data }) => {
                return {
                    data: data.data,
                };
            },
        },
        yup: runRoutineInputValidation,
    },
    search: {
        defaultSort: RunRoutineInputSortBy.DateUpdatedDesc,
        sortBy: RunRoutineInputSortBy,
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
        isPublic: (...rest) => oneIsPublic<RunRoutineInputModelInfo["DbSelect"]>([["runRoutine", "RunRoutine"]], ...rest),
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
