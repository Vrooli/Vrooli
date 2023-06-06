import { runProjectStepValidation } from "@local/shared";
import { RunProjectStepFormat } from "../format/runProjectStep";
import { ModelLogic } from "../types";
import { RunProjectStepModelLogic } from "./types";

const __typename = "RunProjectStep" as const;
const suppFields = [] as const;
export const RunProjectStepModel: ModelLogic<RunProjectStepModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.run_project_step,
    display: {
        label: {
            select: () => ({ id: true, name: true }),
            get: (select) => select.name,
        },
    },
    format: RunProjectStepFormat,
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
