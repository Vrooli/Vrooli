import { runProjectStepValidation } from "@local/validation";
const __typename = "RunProjectStep";
const suppFields = [];
export const RunProjectStepModel = ({
    __typename,
    delegate: (prisma) => prisma.run_project_step,
    display: {
        select: () => ({ id: true, name: true }),
        label: (select) => select.name,
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
            }),
            update: async ({ data, ...rest }) => ({
                id: data.id,
            }),
        },
        yup: runProjectStepValidation,
    },
    validate: {},
});
//# sourceMappingURL=runProjectStep.js.map