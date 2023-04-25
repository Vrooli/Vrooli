import { NodeLinkWhen, NodeLinkWhenCreateInput, NodeLinkWhenUpdateInput, nodeLinkWhenValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, shapeHelper } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { translationShapeHelper } from "../utils";
import { ModelLogic } from "./types";

const __typename = "NodeLinkWhen" as const;
const suppFields = [] as const;
export const NodeLinkWhenModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: NodeLinkWhenCreateInput,
    GqlUpdate: NodeLinkWhenUpdateInput,
    GqlModel: NodeLinkWhen,
    GqlPermission: {},
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.node_link_whenUpsertArgs["create"],
    PrismaUpdate: Prisma.node_link_whenUpsertArgs["update"],
    PrismaModel: Prisma.node_link_whenGetPayload<SelectWrap<Prisma.node_link_whenSelect>>,
    PrismaSelect: Prisma.node_link_whenSelect,
    PrismaWhere: Prisma.node_link_whenWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.node_link,
    // Doesn't make sense to have a displayer for this model
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