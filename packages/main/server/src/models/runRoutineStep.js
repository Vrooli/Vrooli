import { MaxObjects } from "@local/consts";
import { runRoutineStepValidation } from "@local/validation";
import { defaultPermissions } from "../utils";
import { RunRoutineModel } from "./runRoutine";
const __typename = "RunRoutineStep";
const suppFields = [];
export const RunRoutineStepModel = ({
    __typename,
    delegate: (prisma) => prisma.run_routine_step,
    display: {
        select: () => ({ id: true, name: true }),
        label: (select) => select.name,
    },
    format: {
        gqlRelMap: {
            __typename,
            run: "RunRoutine",
            node: "Node",
            subroutine: "Routine",
        },
        prismaRelMap: {
            __typename,
            node: "Node",
            runRoutine: "RunRoutine",
            subroutine: "RoutineVersion",
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, userData }) => {
                return {};
            },
            update: async ({ data, userData }) => {
                return {};
            },
        },
        yup: runRoutineStepValidation,
    },
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            runRoutine: "RunRoutine",
        }),
        permissionResolvers: defaultPermissions,
        profanityFields: ["name"],
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
//# sourceMappingURL=runRoutineStep.js.map