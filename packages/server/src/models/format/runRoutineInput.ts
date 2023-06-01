import { MaxObjects, RunRoutineInput, RunRoutineInputCreateInput, RunRoutineInputSearchInput, RunRoutineInputSortBy, RunRoutineInputUpdateInput, runRoutineInputValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { RoutineVersionInputModel } from ".";
import { selPad } from "../../builders";
import { SelectWrap } from "../../builders/types";
import { PrismaType } from "../../types";
import { defaultPermissions } from "../../utils";
import { RunRoutineModel } from "./runRoutine";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "RunRoutineInput" as const;
export const RunRoutineInputFormat: Formatter<ModelRunRoutineInputLogic> = {
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
        searchFields: {
            createdTimeFrame: true,
            excludeIds: true,
            routineIds: true,
            standardIds: true,
            updatedTimeFrame: true,
        permissionsSelect: () => ({
            id: true,
            runRoutine: "RunRoutine",
        visibility: {
            private: { runRoutine: { isPrivate: true } },
            public: { runRoutine: { isPrivate: false } },
            owner: (userId) => ({ runRoutine: RunRoutineModel.validate!.visibility.owner(userId) }),
};
