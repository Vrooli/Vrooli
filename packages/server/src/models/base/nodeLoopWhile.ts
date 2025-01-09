import { MaxObjects, nodeLoopWhileValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { useVisibility } from "../../builders/visibilityBuilder";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { translationShapeHelper } from "../../utils/shapes";
import { NodeLoopWhileFormat } from "../formats";
import { NodeLoopModelInfo, NodeLoopModelLogic, NodeLoopWhileModelInfo, NodeLoopWhileModelLogic } from "./types";

const __typename = "NodeLoopWhile" as const;
export const NodeLoopWhileModel: NodeLoopWhileModelLogic = ({
    __typename,
    dbTable: "node_loop_while",
    dbTranslationTable: "node_loop_while_translation",
    // Doesn't make sense to have a displayer for this model
    display: () => ({
        label: {
            select: () => ({ id: true }),
            get: () => "",
        },
    }),
    format: NodeLoopWhileFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                condition: data.condition,
                loop: await shapeHelper({ relation: "loop", relTypes: ["Connect"], isOneToOne: true, objectType: "NodeLoop", parentRelationshipName: "whiles", data, ...rest }),
                translations: await translationShapeHelper({ relTypes: ["Create"], data, ...rest }),

            }),
            update: async ({ data, ...rest }) => ({
                condition: noNull(data.condition),
                translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], data, ...rest }),
            }),
        },
        yup: nodeLoopWhileValidation,
    },
    search: undefined,
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ id: true, loop: "NodeLoop" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<NodeLoopModelLogic>("NodeLoop").validate().owner(data?.loop as NodeLoopModelInfo["DbModel"], userId),
        isDeleted: (data, languages) => ModelMap.get<NodeLoopModelLogic>("NodeLoop").validate().isDeleted(data.loop as NodeLoopModelInfo["DbModel"], languages),
        isPublic: (...rest) => oneIsPublic<NodeLoopWhileModelInfo["DbSelect"]>([["loop", "NodeLoop"]], ...rest),
        visibility: {
            own: function getOwn(data) {
                return {
                    loop: useVisibility("NodeLoop", "Own", data),
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    loop: useVisibility("NodeLoop", "OwnOrPublic", data),
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    loop: useVisibility("NodeLoop", "OwnPrivate", data),
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    loop: useVisibility("NodeLoop", "OwnPublic", data),
                };
            },
            public: function getPublic(data) {
                return {
                    loop: useVisibility("NodeLoop", "Public", data),
                };
            },
        },
    }),
});
