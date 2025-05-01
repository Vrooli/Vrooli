import { MaxObjects, runRoutineStepValidation } from "@local/shared";
import { noNull } from "../../builders/noNull.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { oneIsPublic } from "../../utils/oneIsPublic.js";
import { RunRoutineStepFormat } from "../formats.js";
import { ModelMap } from "./index.js";
import { RunRoutineModelInfo, RunRoutineModelLogic, RunRoutineStepModelInfo, RunRoutineStepModelLogic } from "./types.js";

const __typename = "RunRoutineStep" as const;
export const RunRoutineStepModel: RunRoutineStepModelLogic = ({
    __typename,
    dbTable: "run_routine_step",
    display: () => ({
        label: {
            select: () => ({ id: true, name: true }),
            get: (select) => select.name,
        },
    }),
    format: RunRoutineStepFormat,
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
                    subroutineInId: data.subroutineInId,
                    timeElapsed,
                    runRoutine: await shapeHelper({ relation: "runRoutine", relTypes: ["Connect"], isOneToOne: true, objectType: "RunRoutine", parentRelationshipName: "steps", data, ...rest }),
                    subroutine: await shapeHelper({ relation: "subroutine", relTypes: ["Connect"], isOneToOne: true, objectType: "RoutineVersion", parentRelationshipName: "runSteps", data, ...rest }),
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
            runRoutine: "RunRoutine",
        }),
        permissionResolvers: defaultPermissions,
        profanityFields: ["name"],
        owner: (data, userId) => ModelMap.get<RunRoutineModelLogic>("RunRoutine").validate().owner(data?.runRoutine as RunRoutineModelInfo["DbModel"], userId),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<RunRoutineStepModelInfo["DbSelect"]>([["runRoutine", "RunRoutine"]], ...rest),
        visibility: {
            own: function getOwn(data) {
                return {
                    runRoutine: useVisibility("RunRoutine", "Own", data),
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    runRoutine: useVisibility("RunRoutine", "OwnOrPublic", data),
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    runRoutine: useVisibility("RunRoutine", "OwnPrivate", data),
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    runRoutine: useVisibility("RunRoutine", "OwnPublic", data),
                };
            },
            public: function getPublic(data) {
                return {
                    runRoutine: useVisibility("RunRoutine", "Public", data),
                };
            },
        },
    }),
});
