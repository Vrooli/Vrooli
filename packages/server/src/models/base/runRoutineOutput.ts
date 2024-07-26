import { MaxObjects, RunRoutineOutputSortBy, runRoutineOutputValidation } from "@local/shared";
import { ModelMap } from ".";
import { shapeHelper } from "../../builders/shapeHelper";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { RunRoutineOutputFormat } from "../formats";
import { RoutineVersionOutputModelInfo, RoutineVersionOutputModelLogic, RunRoutineModelInfo, RunRoutineModelLogic, RunRoutineOutputModelInfo, RunRoutineOutputModelLogic } from "./types";

const __typename = "RunRoutineOutput" as const;
export const RunRoutineOutputModel: RunRoutineOutputModelLogic = ({
    __typename,
    dbTable: "run_routine_output",
    display: () => ({
        label: {
            select: () => ({
                id: true,
                output: { select: ModelMap.get<RoutineVersionOutputModelLogic>("RoutineVersionOutput").display().label.select() },
                runRoutine: { select: ModelMap.get<RunRoutineModelLogic>("RunRoutine").display().label.select() },
            }),
            // Label combines runRoutine's label and output's label
            get: (select, languages) => {
                const runRoutineLabel = ModelMap.get<RunRoutineModelLogic>("RunRoutine").display().label.get(select.runRoutine as RunRoutineModelInfo["PrismaModel"], languages);
                const outputLabel = ModelMap.get<RoutineVersionOutputModelLogic>("RoutineVersionOutput").display().label.get(select.output as RoutineVersionOutputModelInfo["PrismaModel"], languages);
                if (runRoutineLabel.length > 0) {
                    return `${runRoutineLabel} - ${outputLabel}`;
                }
                return outputLabel;
            },
        },
    }),
    format: RunRoutineOutputFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => {
                return {
                    id: data.id,
                    data: data.data,
                    output: await shapeHelper({ relation: "output", relTypes: ["Connect"], isOneToOne: true, objectType: "RoutineVersionOutput", parentRelationshipName: "runOutputs", data, ...rest }),
                    runRoutine: await shapeHelper({ relation: "runRoutine", relTypes: ["Connect"], isOneToOne: true, objectType: "RunRoutine", parentRelationshipName: "outputs", data, ...rest }),
                };
            },
            update: async ({ data }) => {
                return {
                    data: data.data,
                };
            },
        },
        yup: runRoutineOutputValidation,
    },
    search: {
        defaultSort: RunRoutineOutputSortBy.DateUpdatedDesc,
        sortBy: RunRoutineOutputSortBy,
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
        owner: (data, userId) => ModelMap.get<RunRoutineModelLogic>("RunRoutine").validate().owner(data?.runRoutine as RunRoutineModelInfo["PrismaModel"], userId),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<RunRoutineOutputModelInfo["PrismaSelect"]>([["runRoutine", "RunRoutine"]], ...rest),
        visibility: {
            private: function getVisibilityPrivate() {
                return {
                    runRoutine: { isPrivate: true },
                };
            },
            public: function getVisibilityPublic() {
                return {
                    runRoutine: { isPrivate: false },
                };
            },
            owner: (userId) => ({ runRoutine: ModelMap.get<RunRoutineModelLogic>("RunRoutine").validate().visibility.owner(userId) }),
        },
    }),
});
