import { nodeLinkWhenValidation } from "@local/validation";
import { noNull, shapeHelper } from "../builders";
import { translationShapeHelper } from "../utils";
const __typename = "NodeLinkWhen";
const suppFields = [];
export const NodeLinkWhenModel = ({
    __typename,
    delegate: (prisma) => prisma.node_link,
    display: {
        select: () => ({ id: true }),
        label: () => "",
    },
    format: {
        gqlRelMap: {
            __typename,
        },
        prismaRelMap: {
            __typename,
            link: "NodeLink",
        },
        countFields: {},
    },
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
//# sourceMappingURL=nodeLinkWhen.js.map