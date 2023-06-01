import { MaxObjects, RunRoutineSearchInput, RunRoutineSortBy, RunRoutineStep, RunRoutineStepCreateInput, RunRoutineStepUpdateInput, runRoutineStepValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { SelectWrap } from "../../builders/types";
import { PrismaType } from "../../types";
import { defaultPermissions } from "../../utils";
import { RunRoutineModel } from "./runRoutine";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "RunRoutineStep" as const;
export const RunRoutineStepFormat: Formatter<ModelRunRoutineStepLogic> = {
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
        permissionsSelect: () => ({
            id: true,
            runRoutine: "RunRoutine",
        visibility: {
            private: { runRoutine: { isPrivate: true } },
            public: { runRoutine: { isPrivate: false } },
            owner: (userId) => ({ runRoutine: RunRoutineModel.validate!.visibility.owner(userId) }),
};
