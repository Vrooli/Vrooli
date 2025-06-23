import { MaxObjects, runRoutineStepValidation } from "@vrooli/shared";
import { noNull } from "../../builders/noNull.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { oneIsPublic } from "../../utils/oneIsPublic.js";
import { defaultPermissions } from "../../validators/permissions.js";
import { RunStepFormat } from "../formats.js";
import { ModelMap } from "./index.js";
import { type RunModelInfo, type RunModelLogic, type RunStepModelInfo, type RunStepModelLogic } from "./types.js";

const __typename = "RunStep" as const;
export const RunStepModel: RunStepModelLogic = ({
    __typename,
    dbTable: "run_step",
    display: () => ({
        label: {
            select: () => ({ id: true, name: true }),
            get: (select) => select.name,
        },
    }),
    format: RunStepFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => {
                let contextSwitches = noNull(data.contextSwitches);
                if (contextSwitches !== undefined) contextSwitches = Math.max(contextSwitches, 0);
                let timeElapsed = noNull(data.timeElapsed);
                if (timeElapsed !== undefined) timeElapsed = Math.max(timeElapsed, 0);
                return {
                    id: BigInt(data.id),
                    complexity: data.complexity,
                    contextSwitches,
                    name: data.name,
                    nodeId: data.nodeId,
                    order: data.order,
                    status: noNull(data.status),
                    resourceInId: data.resourceInId,
                    timeElapsed,
                    run: await shapeHelper({ relation: "runRoutine", relTypes: ["Connect"], isOneToOne: true, objectType: "Run", parentRelationshipName: "steps", data, ...rest }),
                    resourceVersion: await shapeHelper({ relation: "resourceVersion", relTypes: ["Connect"], isOneToOne: true, objectType: "ResourceVersion", parentRelationshipName: "runSteps", data, ...rest }),
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
        yup: runRoutineStepValidation,
    },
    search: undefined,
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            runRoutine: "Run",
        }),
        permissionResolvers: defaultPermissions,
        profanityFields: ["name"],
        owner: (data, userId) => ModelMap.get<RunModelLogic>("Run").validate().owner(data?.run as RunModelInfo["DbModel"], userId),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<RunStepModelInfo["DbSelect"]>([["run", "Run"]], ...rest),
        visibility: {
            own: function getOwn(data) {
                return {
                    run: useVisibility("Run", "Own", data),
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    run: useVisibility("Run", "OwnOrPublic", data),
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    run: useVisibility("Run", "OwnPrivate", data),
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    run: useVisibility("Run", "OwnPublic", data),
                };
            },
            public: function getPublic(data) {
                return {
                    run: useVisibility("Run", "Public", data),
                };
            },
        },
    }),
});
