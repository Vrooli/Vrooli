import { MaxObjects, runRoutineStepValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { RunRoutineStepFormat } from "../formats";
import { RunRoutineModelInfo, RunRoutineModelLogic, RunRoutineStepModelInfo, RunRoutineStepModelLogic } from "./types";

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
                    id: data.id,
                    contextSwitches,
                    name: data.name,
                    order: data.order,
                    status: noNull(data.status),
                    step: data.step,
                    timeElapsed,
                    node: await shapeHelper({ relation: "node", relTypes: ["Connect"], isOneToOne: true, objectType: "Node", parentRelationshipName: "runSteps", data, ...rest }),
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
        owner: (data, userId) => ModelMap.get<RunRoutineModelLogic>("RunRoutine").validate().owner(data?.runRoutine as RunRoutineModelInfo["PrismaModel"], userId),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<RunRoutineStepModelInfo["PrismaSelect"]>([["runRoutine", "RunRoutine"]], ...rest),
        visibility: {
            private: function getVisibilityPrivate() {
                return {
                    runRoutine: { isPrivate: true },
                };
            },
            public: function getVisibilityPublic() {
                return {
                    runRoutine: { isPrivate: false },
                };
            },
            owner: (userId) => ({ runRoutine: ModelMap.get<RunRoutineModelLogic>("RunRoutine").validate().visibility.owner(userId) }),
        },
    }),
});
