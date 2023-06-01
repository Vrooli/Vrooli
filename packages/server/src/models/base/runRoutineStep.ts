import { MaxObjects, runRoutineStepValidation } from "@local/shared";
import { PrismaType } from "../types";
import { defaultPermissions } from "../utils";
import { RunRoutineModel } from "./runRoutine";
import { ModelLogic, RunRoutineStepModelLogic } from "./types";

// const shapeBase = (data: RunRoutineStepCreateInput | RunRoutineStepUpdateInput) => {
//     return {
//         id: data.id,
//         contextSwitches: data.contextSwitches ?? undefined,
//         timeElapsed: data.timeElapsed,
//     }
// }

const __typename = "RunRoutineStep" as const;
const suppFields = [] as const;
export const RunRoutineStepModel: ModelLogic<RunRoutineStepModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.run_routine_step,
    display: {
        label: {
            select: () => ({ id: true, name: true }),
            get: (select) => select.name,
        },
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
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            runRoutine: "RunRoutine",
        }),
        permissionResolvers: defaultPermissions,
        profanityFields: ["name"],
        owner: (data, userId) => RunRoutineModel.validate!.owner(data.runRoutine as any, userId),
        isDeleted: () => false,
        isPublic: (data, languages) => RunRoutineModel.validate!.isPublic(data.runRoutine as any, languages),
        visibility: {
            private: { runRoutine: { isPrivate: true } },
            public: { runRoutine: { isPrivate: false } },
            owner: (userId) => ({ runRoutine: RunRoutineModel.validate!.visibility.owner(userId) }),
        },
    },
});
