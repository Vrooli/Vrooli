import { MaxObjects, RunRoutineInputSortBy, runRoutineInputValidation } from "@local/shared";
import { ModelMap } from ".";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { RunRoutineInputFormat } from "../formats";
import { RoutineVersionInputModelInfo, RoutineVersionInputModelLogic, RunRoutineInputModelInfo, RunRoutineInputModelLogic, RunRoutineModelInfo, RunRoutineModelLogic } from "./types";

const __typename = "RunRoutineInput" as const;
export const RunRoutineInputModel: RunRoutineInputModelLogic = ({
    __typename,
    delegate: (p) => p.run_routine_input,
    display: () => ({
        label: {
            select: () => ({
                id: true,
                input: { select: ModelMap.get<RoutineVersionInputModelLogic>("RoutineVersionInput").display().label.select() },
                runRoutine: { select: ModelMap.get<RunRoutineModelLogic>("RunRoutine").display().label.select() },
            }),
            // Label combines runRoutine's label and input's label
            get: (select, languages) => {
                const runRoutineLabel = ModelMap.get<RunRoutineModelLogic>("RunRoutine").display().label.get(select.runRoutine as RunRoutineModelInfo["PrismaModel"], languages);
                const inputLabel = ModelMap.get<RoutineVersionInputModelLogic>("RoutineVersionInput").display().label.get(select.input as RoutineVersionInputModelInfo["PrismaModel"], languages);
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
            create: async ({ data }) => {
                return {
                    // id: data.id,
                    // data: data.data,
                    // input: { connect: { id: data.inputId } },
                } as any;
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
            routineIds: true,
            standardIds: true,
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
        isPublic: (...rest) => oneIsPublic<RunRoutineInputModelInfo["PrismaSelect"]>([["runRoutine", "RunRoutine"]], ...rest),
        visibility: {
            private: { runRoutine: { isPrivate: true } },
            public: { runRoutine: { isPrivate: false } },
            owner: (userId) => ({ runRoutine: ModelMap.get<RunRoutineModelLogic>("RunRoutine").validate().visibility.owner(userId) }),
        },
    }),
});
