import { MaxObjects, nodeLinkValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { useVisibility } from "../../builders/visibilityBuilder";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { NodeLinkFormat } from "../formats";
import { NodeLinkModelInfo, NodeLinkModelLogic, NodeModelInfo, NodeModelLogic, RoutineVersionModelInfo, RoutineVersionModelLogic } from "./types";

const __typename = "NodeLink" as const;
export const NodeLinkModel: NodeLinkModelLogic = ({
    __typename,
    dbTable: "node_link",
    display: () => ({
        label: {
            select: () => ({
                id: true,
                from: { select: ModelMap.get<NodeModelLogic>("Node").display().label.select() },
                to: { select: ModelMap.get<NodeModelLogic>("Node").display().label.select() },
            }),
            // Label combines from and to labels
            get: (select, languages) => {
                return `${ModelMap.get<NodeModelLogic>("Node").display().label.get(select.from as NodeModelInfo["PrismaModel"], languages)} -> ${ModelMap.get<NodeModelLogic>("Node").display().label.get(select.to as NodeModelInfo["PrismaModel"], languages)}`;
            },
        },
    }),
    format: NodeLinkFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                operation: noNull(data.operation),
                from: await shapeHelper({ relation: "from", relTypes: ["Connect"], isOneToOne: true, objectType: "Node", parentRelationshipName: "next", data, ...rest }),
                to: await shapeHelper({ relation: "to", relTypes: ["Connect"], isOneToOne: true, objectType: "Node", parentRelationshipName: "previous", data, ...rest }),
                routineVersion: await shapeHelper({ relation: "routineVersion", relTypes: ["Connect"], isOneToOne: true, objectType: "RoutineVersion", parentRelationshipName: "nodeLinks", data, ...rest }),
                whens: await shapeHelper({ relation: "whens", relTypes: ["Create"], isOneToOne: false, objectType: "NodeLinkWhen", parentRelationshipName: "link", data, ...rest }),

            }),
            update: async ({ data, ...rest }) => ({
                operation: noNull(data.operation),
                from: await shapeHelper({ relation: "from", relTypes: ["Connect", "Disconnect"], isOneToOne: true, objectType: "Node", parentRelationshipName: "next", data, ...rest }),
                to: await shapeHelper({ relation: "to", relTypes: ["Connect", "Disconnect"], isOneToOne: true, objectType: "Node", parentRelationshipName: "previous", data, ...rest }),
                when: await shapeHelper({ relation: "whens", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "NodeLinkWhen", parentRelationshipName: "link", data, ...rest }),
            }),
        },
        yup: nodeLinkValidation,
    },
    search: undefined,
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ id: true, routineVersion: "RoutineVersion" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<RoutineVersionModelLogic>("RoutineVersion").validate().owner(data?.routineVersion as RoutineVersionModelInfo["PrismaModel"], userId),
        isDeleted: (data, languages) => ModelMap.get<RoutineVersionModelLogic>("RoutineVersion").validate().isDeleted(data.routineVersion as RoutineVersionModelInfo["PrismaModel"], languages),
        isPublic: (...rest) => oneIsPublic<NodeLinkModelInfo["PrismaSelect"]>([["routineVersion", "RoutineVersion"]], ...rest),
        visibility: {
            private: function getVisibilityPrivate(...params) {
                return {
                    routineVersion: useVisibility("Routine", "private", ...params),
                };
            },
            public: function getVisibilityPublic(...params) {
                return {
                    routineVersion: useVisibility("Routine", "public", ...params),
                };
            },
            owner: (...params) => ({ routineVersion: useVisibility("Routine", "owner", ...params) }),
        },
    }),
});
