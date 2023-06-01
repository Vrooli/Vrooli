import { NodeLinkWhen, NodeLinkWhenCreateInput, NodeLinkWhenUpdateInput, nodeLinkWhenValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, shapeHelper } from "../../builders";
import { SelectWrap } from "../../builders/types";
import { PrismaType } from "../../types";
import { translationShapeHelper } from "../../utils";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "NodeLinkWhen" as const;
export const NodeLinkWhenFormat: Formatter<ModelNodeLinkWhenLogic> = {
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
};
