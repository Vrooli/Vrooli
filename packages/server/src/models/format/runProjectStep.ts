import { RunProjectStep, RunProjectStepCreateInput, RunProjectStepUpdateInput, runProjectStepValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { SelectWrap } from "../../builders/types";
import { PrismaType } from "../../types";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "RunProjectStep" as const;
export const RunProjectStepFormat: Formatter<ModelRunProjectStepLogic> = {
        gqlRelMap: {
            __typename,
            directory: "ProjectVersionDirectory",
            run: "RunProject",
        },
        prismaRelMap: {
            __typename,
            directory: "ProjectVersionDirectory",
            runProject: "RunProject",
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                //TODO
            } as any),
            update: async ({ data, ...rest }) => ({
                id: data.id,
                //TODO
            } as any),
};
