import { runProjectStepValidation } from "@local/shared";
import { ModelMap } from ".";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { RunProjectStepFormat } from "../formats";
import { RunProjectModelInfo, RunProjectModelLogic, RunProjectStepModelInfo, RunProjectStepModelLogic } from "./types";

const __typename = "RunProjectStep" as const;
export const RunProjectStepModel: RunProjectStepModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.run_project_step,
    display: () => ({
        label: {
            select: () => ({ id: true, name: true }),
            get: (select) => select.name,
        },
    }),
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
    validate: () => ({
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<RunProjectStepModelInfo["PrismaSelect"]>([["runProject", "RunProject"]], ...rest),
        isTransferable: false,
        maxObjects: 100000,
        owner: (data, userId) => ModelMap.get<RunProjectModelLogic>("RunProject").validate().owner(data?.runProject as RunProjectModelInfo["PrismaModel"], userId),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({ id: true, runProject: "RunProject" }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({ runProject: ModelMap.get<RunProjectModelLogic>("RunProject").validate().visibility.owner(userId) }),
        },
    }),
});
