import { MaxObjects, nodeLinkWhenValidation } from "@local/shared";
import { noNull, shapeHelper } from "../../builders";
import { defaultPermissions } from "../../utils";
import { translationShapeHelper } from "../../utils/shapes";
import { NodeLinkWhenFormat } from "../formats";
import { ModelLogic } from "../types";
import { NodeLinkModel } from "./nodeLink";
import { NodeLinkModelLogic, NodeLinkWhenModelLogic } from "./types";

const __typename = "NodeLinkWhen" as const;
const suppFields = [] as const;
export const NodeLinkWhenModel: ModelLogic<NodeLinkWhenModelLogic, typeof suppFields> = ({
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
        owner: (data, userId) => NodeLinkModel.validate.owner(data?.link as NodeLinkModelLogic["PrismaModel"], userId),
        isDeleted: (data, languages) => NodeLinkModel.validate.isDeleted(data.link as NodeLinkModelLogic["PrismaModel"], languages),
        isPublic: (data, getParentInfo, languages) => NodeLinkModel.validate.isPublic((data.link ?? getParentInfo(data.id, "NodeLink")) as NodeLinkModelLogic["PrismaModel"], getParentInfo, languages),
        visibility: {
            private: { link: NodeLinkModel.validate.visibility.private },
            public: { link: NodeLinkModel.validate.visibility.public },
            owner: (userId) => ({ link: NodeLinkModel.validate.visibility.owner(userId) }),
        },
    },
});
