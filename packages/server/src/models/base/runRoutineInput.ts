import { MaxObjects, RunRoutineInputSortBy, runRoutineInputValidation } from "@local/shared";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { RunRoutineInputFormat } from "../formats";
import { ModelLogic } from "../types";
import { RoutineVersionInputModel } from "./routineVersionInput";
import { RunRoutineModel } from "./runRoutine";
import { RoutineVersionInputModelLogic, RunRoutineInputModelLogic, RunRoutineModelLogic } from "./types";

const __typename = "RunRoutineInput" as const;
const suppFields = [] as const;
export const RunRoutineInputModel: ModelLogic<RunRoutineInputModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.run_routine_input,
    display: {
        label: {
            select: () => ({
                id: true,
                input: { select: RoutineVersionInputModel.display.label.select() },
                runRoutine: { select: RunRoutineModel.display.label.select() },
            }),
            // Label combines runRoutine's label and input's label
            get: (select, languages) => {
                const runRoutineLabel = RunRoutineModel.display.label.get(select.runRoutine as RunRoutineModelLogic["PrismaModel"], languages);
                const inputLabel = RoutineVersionInputModel.display.label.get(select.input as RoutineVersionInputModelLogic["PrismaModel"], languages);
                if (runRoutineLabel.length > 0) {
                    return `${runRoutineLabel} - ${inputLabel}`;
                }
                return inputLabel;
            },
        },
    },
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
        searchStringQuery: () => ({ runRoutine: RunRoutineModel.search.searchStringQuery() }),
    },
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            runRoutine: "RunRoutine",
        }),
        permissionResolvers: defaultPermissions,
        profanityFields: ["data"],
        owner: (data, userId) => RunRoutineModel.validate.owner(data?.runRoutine as RunRoutineModelLogic["PrismaModel"], userId),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<RunRoutineInputModelLogic["PrismaSelect"]>([["runRoutine", "RunRoutine"]], ...rest),
        visibility: {
            private: { runRoutine: { isPrivate: true } },
            public: { runRoutine: { isPrivate: false } },
            owner: (userId) => ({ runRoutine: RunRoutineModel.validate.visibility.owner(userId) }),
        },
    },
});
