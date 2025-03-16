import { MaxObjects, runProjectStepValidation } from "@local/shared";
import { noNull } from "../../builders/noNull.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { oneIsPublic } from "../../utils/oneIsPublic.js";
import { RunProjectStepFormat } from "../formats.js";
import { ModelMap } from "./index.js";
import { RunProjectModelInfo, RunProjectModelLogic, RunProjectStepModelInfo, RunProjectStepModelLogic } from "./types.js";

const __typename = "RunProjectStep" as const;
export const RunProjectStepModel: RunProjectStepModelLogic = ({
    __typename,
    dbTable: "run_project_step",
    display: () => ({
        label: {
            select: () => ({ id: true, name: true }),
            get: (select) => select.name,
        },
    }),
    format: RunProjectStepFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => {
                let contextSwitches = noNull(data.contextSwitches);
                if (contextSwitches !== undefined) contextSwitches = Math.max(contextSwitches, 0);
                let timeElapsed = noNull(data.timeElapsed);
                if (timeElapsed !== undefined) timeElapsed = Math.max(timeElapsed, 0);
                return {
                    id: data.id,
                    complexity: data.complexity,
                    contextSwitches,
                    directoryInId: data.directoryInId,
                    name: data.name,
                    order: data.order,
                    status: noNull(data.status),
                    timeElapsed,
                    directory: await shapeHelper({ relation: "directory", relTypes: ["Connect"], isOneToOne: true, objectType: "ProjectVersionDirectory", parentRelationshipName: "runProjectSteps", data, ...rest }),
                    runProject: await shapeHelper({ relation: "runProject", relTypes: ["Connect"], isOneToOne: true, objectType: "RunProject", parentRelationshipName: "steps", data, ...rest }),
                };
            },
            update: async ({ data }) => {
                let contextSwitches = noNull(data.contextSwitches);
                if (contextSwitches !== undefined) contextSwitches = Math.max(contextSwitches, 0);
                let timeElapsed = noNull(data.timeElapsed);
                if (timeElapsed !== undefined) timeElapsed = Math.max(timeElapsed, 0);
                return {
                    contextSwitches,
                    status: noNull(data.status),
                    timeElapsed,
                };
            },
        },
        yup: runProjectStepValidation,
    },
    search: undefined,
    validate: () => ({
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<RunProjectStepModelInfo["DbSelect"]>([["runProject", "RunProject"]], ...rest),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => ModelMap.get<RunProjectModelLogic>("RunProject").validate().owner(data?.runProject as RunProjectModelInfo["DbModel"], userId),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({ id: true, runProject: "RunProject" }),
        visibility: {
            own: function getOwn(data) {
                return {
                    runProject: useVisibility("RunProject", "Own", data),
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    runProject: useVisibility("RunProject", "OwnOrPublic", data),
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    runProject: useVisibility("RunProject", "OwnPrivate", data),
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    runProject: useVisibility("RunProject", "OwnPublic", data),
                };
            },
            public: function getPublic(data) {
                return {
                    runProject: useVisibility("RunProject", "Public", data),
                };
            },
        },
    }),
});
