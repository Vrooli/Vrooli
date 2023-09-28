import { runProjectStepValidation } from "@local/shared";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { RunProjectStepFormat } from "../formats";
import { ModelLogic } from "../types";
import { RunProjectModel } from "./runProject";
import { RunProjectModelLogic, RunProjectStepModelLogic } from "./types";

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
    search: undefined,
    validate: {
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<RunProjectStepModelLogic["PrismaSelect"]>([["runProject", "RunProject"]], ...rest),
        isTransferable: false,
        maxObjects: 100000,
        owner: (data, userId) => RunProjectModel.validate.owner(data?.runProject as RunProjectModelLogic["PrismaModel"], userId),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({ id: true, runProject: "RunProject" }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({ runProject: RunProjectModel.validate.visibility.owner(userId) }),
        },
    },
});
