import { MaxObjects, nodeLinkWhenValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { translationShapeHelper } from "../../utils/shapes";
import { NodeLinkWhenFormat } from "../formats";
import { NodeLinkModelInfo, NodeLinkModelLogic, NodeLinkWhenModelInfo, NodeLinkWhenModelLogic } from "./types";

const __typename = "NodeLinkWhen" as const;
export const NodeLinkWhenModel: NodeLinkWhenModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.node_link,
    // Doesn't make sense to have a displayer for this model
    display: {
        label: {
            select: () => ({ id: true }),
            get: () => "",
        },
    },
    format: NodeLinkWhenFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                condition: data.condition,
                ...(await shapeHelper({ relation: "link", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "NodeLink", parentRelationshipName: "link", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                condition: noNull(data.condition),
                ...(await shapeHelper({ relation: "link", relTypes: ["Connect"], isOneToOne: true, isRequired: false, objectType: "NodeLink", parentRelationshipName: "link", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, data, ...rest })),
            }),
        },
        yup: nodeLinkWhenValidation,
    },
    search: undefined,
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ id: true, link: "NodeLink" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<NodeLinkModelLogic>("NodeLink").validate.owner(data?.link as NodeLinkModelInfo["PrismaModel"], userId),
        isDeleted: (data, languages) => ModelMap.get<NodeLinkModelLogic>("NodeLink").validate.isDeleted(data.link as NodeLinkModelInfo["PrismaModel"], languages),
        isPublic: (...rest) => oneIsPublic<NodeLinkWhenModelInfo["PrismaSelect"]>([["link", "NodeLink"]], ...rest),
        visibility: {
            private: { link: ModelMap.get<NodeLinkModelLogic>("NodeLink").validate.visibility.private },
            public: { link: ModelMap.get<NodeLinkModelLogic>("NodeLink").validate.visibility.public },
            owner: (userId) => ({ link: ModelMap.get<NodeLinkModelLogic>("NodeLink").validate.visibility.owner(userId) }),
        },
    },
});
