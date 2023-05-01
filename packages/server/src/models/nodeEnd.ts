import { NodeEnd, NodeEndCreateInput, NodeEndUpdateInput, nodeEndValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, selPad, shapeHelper } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { nodeEndNextShapeHelper } from "../utils";
import { NodeModel } from "./node";
import { ModelLogic } from "./types";

const __typename = "NodeEnd" as const;
const suppFields = [] as const;
export const NodeEndModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: NodeEndCreateInput,
    GqlUpdate: NodeEndUpdateInput,
    GqlModel: NodeEnd,
    GqlPermission: {},
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.node_endUpsertArgs["create"],
    PrismaUpdate: Prisma.node_endUpsertArgs["update"],
    PrismaModel: Prisma.node_endGetPayload<SelectWrap<Prisma.node_endSelect>>,
    PrismaSelect: Prisma.node_endSelect,
    PrismaWhere: Prisma.node_endWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.node_end,
    display: {
        select: () => ({ id: true, node: selPad(NodeModel.display.select) }),
        label: (select, languages) => NodeModel.display.label(select.node as any, languages),
    },
    format: {
        gqlRelMap: {
            __typename,
            suggestedNextRoutineVersions: "RoutineVersion",
        },
        prismaRelMap: {
            __typename,
            suggestedNextRoutineVersions: "RoutineVersion",
            node: "Node",
        },
        joinMap: { suggestedNextRoutineVersions: "toRoutineVersion" },
        countFields: {},
    },
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
});
