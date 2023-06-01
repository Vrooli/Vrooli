import { NodeLink, NodeLinkCreateInput, NodeLinkUpdateInput, nodeLinkValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, selPad, shapeHelper } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { NodeModel } from "./node";
import { ModelLogic } from "./types";

const __typename = "NodeLink" as const;
const suppFields = [] as const;
export const NodeLinkModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: NodeLinkCreateInput,
    GqlUpdate: NodeLinkUpdateInput,
    GqlModel: NodeLink,
    GqlPermission: object,
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.node_linkUpsertArgs["create"],
    PrismaUpdate: Prisma.node_linkUpsertArgs["update"],
    PrismaModel: Prisma.node_linkGetPayload<SelectWrap<Prisma.node_linkSelect>>,
    PrismaSelect: Prisma.node_linkSelect,
    PrismaWhere: Prisma.node_linkWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.node_link,
    display: {
        label: {
            select: () => ({
                id: true,
                from: selPad(NodeModel.display.label.select),
                to: selPad(NodeModel.display.label.select),
            }),
            // Label combines from and to labels
            get: (select, languages) => {
                return `${NodeModel.display.label.get(select.from as any, languages)} -> ${NodeModel.display.label.get(select.to as any, languages)}`;
            },
        },
    },
    format: {
        gqlRelMap: {
            __typename,
            whens: "NodeLinkWhen",
        },
        prismaRelMap: {
            __typename,
            from: "Node",
            to: "Node",
            routineVersion: "RoutineVersion",
            whens: "NodeLinkWhen",
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                operation: noNull(data.operation),
                ...(await shapeHelper({ relation: "from", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "Node", parentRelationshipName: "next", data, ...rest })),
                ...(await shapeHelper({ relation: "to", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "Node", parentRelationshipName: "previous", data, ...rest })),
                ...(await shapeHelper({ relation: "routineVersion", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "RoutineVersion", parentRelationshipName: "nodeLinks", data, ...rest })),
                ...(await shapeHelper({ relation: "whens", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "NodeLinkWhen", parentRelationshipName: "link", data, ...rest })),

            }),
            update: async ({ data, ...rest }) => ({
                operation: noNull(data.operation),
                ...(await shapeHelper({ relation: "from", relTypes: ["Connect", "Disconnect"], isOneToOne: true, isRequired: false, objectType: "Node", parentRelationshipName: "next", data, ...rest })),
                ...(await shapeHelper({ relation: "to", relTypes: ["Connect", "Disconnect"], isOneToOne: true, isRequired: false, objectType: "Node", parentRelationshipName: "previous", data, ...rest })),
                ...(await shapeHelper({ relation: "whens", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, isRequired: false, objectType: "NodeLinkWhen", parentRelationshipName: "link", data, ...rest })),
            }),
        },
        yup: nodeLinkValidation,
    },
});
