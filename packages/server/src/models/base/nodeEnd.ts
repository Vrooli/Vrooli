import { MaxObjects, nodeEndValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { useVisibility } from "../../builders/visibilityBuilder";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { nodeEndNextShapeHelper } from "../../utils/shapes";
import { NodeEndFormat } from "../formats";
import { NodeEndModelInfo, NodeEndModelLogic, NodeModelInfo, NodeModelLogic } from "./types";

const __typename = "NodeEnd" as const;
export const NodeEndModel: NodeEndModelLogic = ({
    __typename,
    dbTable: "node_end",
    display: () => ({
        label: {
            select: () => ({ id: true, node: { select: ModelMap.get<NodeModelLogic>("Node").display().label.select() } }),
            get: (select, languages) => ModelMap.get<NodeModelLogic>("Node").display().label.get(select.node as NodeModelInfo["DbModel"], languages),
        },
    }),
    format: NodeEndFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => {
                return {
                    id: data.id,
                    wasSuccessful: noNull(data.wasSuccessful),
                    node: await shapeHelper({ relation: "node", relTypes: ["Connect"], isOneToOne: true, objectType: "Node", parentRelationshipName: "end", data, ...rest }),
                    suggestedNextRoutineVersions: await nodeEndNextShapeHelper({ relTypes: ["Connect"], data, ...rest }),
                };
            },
            update: async ({ data, ...rest }) => {
                return {
                    wasSuccessful: noNull(data.wasSuccessful),
                    suggestedNextRoutineVersions: await nodeEndNextShapeHelper({ relTypes: ["Connect", "Disconnect"], data, ...rest }),
                };
            },
        },
        yup: nodeEndValidation,
    },
    search: undefined,
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ id: true, node: "Node" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<NodeModelLogic>("Node").validate().owner(data?.node as NodeModelInfo["DbModel"], userId),
        isDeleted: (data, languages) => ModelMap.get<NodeModelLogic>("Node").validate().isDeleted(data.node as NodeModelInfo["DbModel"], languages),
        isPublic: (...rest) => oneIsPublic<NodeEndModelInfo["DbSelect"]>([["node", "Node"]], ...rest),
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
