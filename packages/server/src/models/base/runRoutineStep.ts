import { MaxObjects, runRoutineStepValidation } from "@local/shared";
import { ModelMap } from ".";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { RunRoutineStepFormat } from "../formats";
import { RunRoutineModelInfo, RunRoutineModelLogic, RunRoutineStepModelInfo, RunRoutineStepModelLogic } from "./types";

// const shapeBase = (data: RunRoutineStepCreateInput | RunRoutineStepUpdateInput) => {
//     return {
//         id: data.id,
//         contextSwitches: data.contextSwitches ?? undefined,
//         timeElapsed: data.timeElapsed,
//     }
// }

const __typename = "RunRoutineStep" as const;
export const RunRoutineStepModel: RunRoutineStepModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.run_routine_step,
    display: () => ({
        label: {
            select: () => ({ id: true, name: true }),
            get: (select) => select.name,
        },
    }),
    format: RunRoutineStepFormat,
    mutate: {
        shape: {
            create: async ({ data, userData }) => {
                return {
                    // ...shapeBase(data),
                    // nodeId: data.nodeId,
                    // subroutineVersionId: data.subroutineVersionId,
                    // order: data.order,
                    // status: RunRoutineStepStatus.InProgress,
                    // step: data.step,
                    // name: data.name,
                } as any;
            },
            update: async ({ data, userData }) => {
                return {
                    // ...shapeBase(data),
                    // status: data.status ?? undefined,
                } as any;
            },
        },
        yup: runRoutineStepValidation,
    },
    search: undefined,
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            runRoutine: "RunRoutine",
        }),
        permissionResolvers: defaultPermissions,
        profanityFields: ["name"],
        owner: (data, userId) => ModelMap.get<RunRoutineModelLogic>("RunRoutine").validate().owner(data?.runRoutine as RunRoutineModelInfo["PrismaModel"], userId),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<RunRoutineStepModelInfo["PrismaSelect"]>([["runRoutine", "RunRoutine"]], ...rest),
        visibility: {
            private: { runRoutine: { isPrivate: true } },
            public: { runRoutine: { isPrivate: false } },
            owner: (userId) => ({ runRoutine: ModelMap.get<RunRoutineModelLogic>("RunRoutine").validate().visibility.owner(userId) }),
        },
    }),
});
