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
    dbTable: "node_link_when",
    dbTranslationTable: "node_link_when_translation",
    // Doesn't make sense to have a displayer for this model
    display: () => ({
        label: {
            select: () => ({ id: true }),
            get: () => "",
        },
    }),
    format: NodeLinkWhenFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                condition: data.condition,
                link: await shapeHelper({ relation: "link", relTypes: ["Connect"], isOneToOne: true, objectType: "NodeLink", parentRelationshipName: "link", data, ...rest }),
                translations: await translationShapeHelper({ relTypes: ["Create"], data, ...rest }),
            }),
            update: async ({ data, ...rest }) => ({
                condition: noNull(data.condition),
                link: await shapeHelper({ relation: "link", relTypes: ["Connect"], isOneToOne: true, objectType: "NodeLink", parentRelationshipName: "link", data, ...rest }),
                translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], data, ...rest }),
            }),
        },
        yup: nodeLinkWhenValidation,
    },
    search: undefined,
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ id: true, link: "NodeLink" }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => ModelMap.get<NodeLinkModelLogic>("NodeLink").validate().owner(data?.link as NodeLinkModelInfo["PrismaModel"], userId),
        isDeleted: (data, languages) => ModelMap.get<NodeLinkModelLogic>("NodeLink").validate().isDeleted(data.link as NodeLinkModelInfo["PrismaModel"], languages),
        isPublic: (...rest) => oneIsPublic<NodeLinkWhenModelInfo["PrismaSelect"]>([["link", "NodeLink"]], ...rest),
        visibility: {
            private: function getVisibilityPrivate(...params) {
                return {
                    link: ModelMap.get<NodeLinkModelLogic>("NodeLink").validate().visibility.private(...params),
                };
            },
            public: function getVisibilityPublic(...params) {
                return {
                    link: ModelMap.get<NodeLinkModelLogic>("NodeLink").validate().visibility.public(...params),
                };
            },
            owner: (userId) => ({ link: ModelMap.get<NodeLinkModelLogic>("NodeLink").validate().visibility.owner(userId) }),
        },
    }),
});
