import { nodeLinkWhenValidation } from "@local/shared";
import { noNull, shapeHelper } from "../../builders";
import { translationShapeHelper } from "../../utils";
import { NodeLinkWhenFormat } from "../format/nodeLinkWhen";
import { ModelLogic } from "../types";
import { NodeLinkWhenModelLogic } from "./types";

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
});
