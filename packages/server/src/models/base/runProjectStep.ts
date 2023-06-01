import { runProjectStepValidation } from "@local/shared";
import { PrismaType } from "../types";
import { ModelLogic, RunProjectStepModelLogic } from "./types";

const __typename = "RunProjectStep" as const;
const suppFields = [] as const;
export const RunProjectStepModel: ModelLogic<RunProjectStepModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.run_project_step,
    display: {
        label: {
            select: () => ({ id: true, name: true }),
            get: (select) => select.name,
        },
    },
    format: {
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
        },
        yup: runProjectStepValidation,
    },
    validate: {} as any,
});
