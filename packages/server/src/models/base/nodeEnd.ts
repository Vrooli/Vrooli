import { MaxObjects, nodeEndValidation } from "@local/shared";
import { noNull, shapeHelper } from "../../builders";
import { defaultPermissions } from "../../utils";
import { nodeEndNextShapeHelper } from "../../utils/shapes";
import { NodeEndFormat } from "../formats";
import { ModelLogic } from "../types";
import { NodeModel } from "./node";
import { NodeEndModelLogic, NodeModelLogic } from "./types";

const __typename = "NodeEnd" as const;
const suppFields = [] as const;
export const NodeEndModel: ModelLogic<NodeEndModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.node_end,
    display: {
        label: {
            select: () => ({ id: true, node: { select: NodeModel.display.label.select() } }),
            get: (select, languages) => NodeModel.display.label.get(select.node as NodeModelLogic["PrismaModel"], languages),
        },
    },
    format: NodeEndFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => {
                return {
                    id: data.id,
                    wasSuccessful: noNull(data.wasSuccessful),
                    ...(await shapeHelper({ relation: "node", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "Node", parentRelationshipName: "end", data, ...rest })),
                    ...(await nodeEndNextShapeHelper({ relTypes: ["Connect"], data, ...rest })),
                };
            },
            update: async ({ data, ...rest }) => {
                return {
                    wasSuccessful: noNull(data.wasSuccessful),
                    ...(await nodeEndNextShapeHelper({ relTypes: ["Connect", "Disconnect"], data, ...rest })),
                };
            },
        },
        yup: nodeEndValidation,
    },
    search: undefined,
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ id: true, node: "Node" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => NodeModel.validate.owner(data?.node as NodeModelLogic["PrismaModel"], userId),
        isDeleted: (data, languages) => NodeModel.validate.isDeleted(data.node as NodeModelLogic["PrismaModel"], languages),
        isPublic: (data, getParentInfo, languages) => NodeModel.validate.isPublic((data.node ?? getParentInfo(data.id, "Node")) as NodeModelLogic["PrismaModel"], getParentInfo, languages),
        visibility: {
            private: { node: NodeModel.validate.visibility.private },
            public: { node: NodeModel.validate.visibility.public },
            owner: (userId) => ({ node: NodeModel.validate.visibility.owner(userId) }),
        },
    },
});
