import { MaxObjects, RunRoutineInputSortBy } from "@local/consts";
import { runRoutineInputValidation } from "@local/validation";
import { RoutineVersionInputModel } from ".";
import { selPad } from "../builders";
import { defaultPermissions } from "../utils";
import { RunRoutineModel } from "./runRoutine";
const __typename = "RunRoutineInput";
const suppFields = [];
export const RunRoutineInputModel = ({
    __typename,
    delegate: (prisma) => prisma.run_routine_input,
    display: {
        select: () => ({
            id: true,
            input: selPad(RoutineVersionInputModel.display.select),
            runRoutine: selPad(RunRoutineModel.display.select),
        }),
        label: (select, languages) => {
            const runRoutineLabel = RunRoutineModel.display.label(select.runRoutine, languages);
            const inputLabel = RoutineVersionInputModel.display.label(select.input, languages);
            if (runRoutineLabel.length > 0) {
                return `${runRoutineLabel} - ${inputLabel}`;
            }
            return inputLabel;
        },
    },
    format: {
        gqlRelMap: {
            __typename,
            input: "RoutineVersionInput",
            runRoutine: "RunRoutine",
        },
        prismaRelMap: {
            __typename,
            input: "RunRoutineInput",
            runRoutine: "RunRoutine",
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data }) => {
                return {};
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
        owner: (data, userId) => RunRoutineModel.validate.owner(data.runRoutine, userId),
        isDeleted: () => false,
        isPublic: (data, languages) => RunRoutineModel.validate.isPublic(data.runRoutine, languages),
        visibility: {
            private: { runRoutine: { isPrivate: true } },
            public: { runRoutine: { isPrivate: false } },
            owner: (userId) => ({ runRoutine: RunRoutineModel.validate.visibility.owner(userId) }),
        },
    },
});
//# sourceMappingURL=runRoutineInput.js.map