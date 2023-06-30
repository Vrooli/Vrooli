import { runProjectStepValidation } from "@local/shared";
import { defaultPermissions } from "../../utils";
import { RunProjectStepFormat } from "../format/runProjectStep";
import { ModelLogic } from "../types";
import { RunProjectModel } from "./runProject";
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
    validate: {
        isDeleted: () => false,
        isPublic: (data, languages) => RunProjectModel.validate.isPublic(data.runProject as any, languages),
        isTransferable: false,
        maxObjects: 100000,
        owner: (data, userId) => RunProjectModel.validate.owner(data.runProject as any, userId),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({ id: true, runProject: "RunProject" }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({ runProject: RunProjectModel.validate.visibility.owner(userId) }),
        },
    },
});
