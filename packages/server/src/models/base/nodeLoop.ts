import { MaxObjects, nodeLoopValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { useVisibility } from "../../builders/visibilityBuilder";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { NodeLoopFormat } from "../formats";
import { NodeLoopModelInfo, NodeLoopModelLogic, NodeModelInfo, NodeModelLogic } from "./types";

const __typename = "NodeLoop" as const;
export const NodeLoopModel: NodeLoopModelLogic = ({
    __typename,
    dbTable: "node_loop",
    // Doesn't make sense to have a displayer for this model
    display: () => ({
        label: {
            select: () => ({ id: true }),
            get: () => "",
        },
    }),
    format: NodeLoopFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                loops: noNull(data.loops),
                maxLoops: noNull(data.maxLoops),
                operation: noNull(data.operation),
                node: await shapeHelper({ relation: "node", relTypes: ["Connect"], isOneToOne: true, objectType: "Node", parentRelationshipName: "loop", data, ...rest }),
                whiles: await shapeHelper({ relation: "whiles", relTypes: ["Create"], isOneToOne: false, objectType: "NodeLoopWhile", parentRelationshipName: "loop", data, ...rest }),
            }),
            update: async ({ data, ...rest }) => ({
                loops: noNull(data.loops),
                maxLoops: noNull(data.maxLoops),
                operation: noNull(data.operation),
                node: await shapeHelper({ relation: "node", relTypes: ["Connect"], isOneToOne: true, objectType: "Node", parentRelationshipName: "loop", data, ...rest }),
                whiles: await shapeHelper({ relation: "whiles", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "NodeLoopWhile", parentRelationshipName: "loop", data, ...rest }),
            }),
        },
        yup: nodeLoopValidation,
    },
    search: undefined,
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ id: true, node: "Node" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<NodeModelLogic>("Node").validate().owner(data?.node as NodeModelInfo["DbModel"], userId),
        isDeleted: (data, languages) => ModelMap.get<NodeModelLogic>("Node").validate().isDeleted(data.node as NodeModelInfo["DbModel"], languages),
        isPublic: (...rest) => oneIsPublic<NodeLoopModelInfo["DbSelect"]>([["node", "Node"]], ...rest),
        visibility: {
            own: function getOwn(data) {
                return {
                    node: useVisibility("Node", "Own", data),
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    node: useVisibility("Node", "OwnOrPublic", data),
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    node: useVisibility("Node", "OwnPrivate", data),
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    node: useVisibility("Node", "OwnPublic", data),
                };
            },
            public: function getPublic(data) {
                return {
                    node: useVisibility("Node", "Public", data),
                };
            },
        },
    }),
});
