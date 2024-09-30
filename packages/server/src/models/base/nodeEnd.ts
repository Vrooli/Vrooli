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
            get: (select, languages) => ModelMap.get<NodeModelLogic>("Node").display().label.get(select.node as NodeModelInfo["PrismaModel"], languages),
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
        owner: (data, userId) => ModelMap.get<NodeModelLogic>("Node").validate().owner(data?.node as NodeModelInfo["PrismaModel"], userId),
        isDeleted: (data, languages) => ModelMap.get<NodeModelLogic>("Node").validate().isDeleted(data.node as NodeModelInfo["PrismaModel"], languages),
        isPublic: (...rest) => oneIsPublic<NodeEndModelInfo["PrismaSelect"]>([["node", "Node"]], ...rest),
        visibility: {
            private: function getVisibilityPrivate(...params) {
                return {
                    node: useVisibility("Node", "private", ...params),
                };
            },
            public: function getVisibilityPublic(...params) {
                return {
                    node: useVisibility("Node", "public", ...params),
                };
            },
            owner: (...params) => ({ node: useVisibility("Node", "owner", ...params) }),
        },
    }),
});
