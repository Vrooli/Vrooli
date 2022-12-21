import { NodeEnd, NodeEndCreateInput, NodeEndUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { Prisma } from "@prisma/client";
import { NodeModel } from "./node";
import { noNull, padSelect, shapeHelper } from "../builders";
import { SelectWrap } from "../builders/types";
import { nodeEndValidation } from "@shared/validation";
import { nodeEndNextShapeHelper } from "../utils";

const __typename = 'NodeEnd' as const;
const suppFields = [] as const;
export const NodeEndModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: NodeEndCreateInput,
    GqlUpdate: NodeEndUpdateInput,
    GqlModel: NodeEnd,
    GqlPermission: any,
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.node_endUpsertArgs['create'],
    PrismaUpdate: Prisma.node_endUpsertArgs['update'],
    PrismaModel: Prisma.node_endGetPayload<SelectWrap<Prisma.node_endSelect>>,
    PrismaSelect: Prisma.node_endSelect,
    PrismaWhere: Prisma.node_endWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.node_end,
    display: {
        select: () => ({ id: true, node: padSelect(NodeModel.display.select) }),
        label: (select, languages) => NodeModel.display.label(select.node as any, languages),
    },
    format: {
        gqlRelMap: {
            __typename,
            suggestedNextRoutineVersion: 'RoutineVersion',
        },
        prismaRelMap: {
            __typename,
            suggestedNextRoutineVersion: 'RoutineVersion',
            node: 'Node',
        },
        joinMap: { suggestedNextRoutineVersion: 'toRoutineVersion' },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, prisma, userData }) => {
                return {
                    id: data.id,
                    wasSuccessful: noNull(data.wasSuccessful),
                    ...(await shapeHelper({ relation: 'node', relTypes: ['Connect'], isOneToOne: true, isRequired: true, objectType: 'Node', parentRelationshipName: 'nodeEnd', data, prisma, userData })),
                    ...(await nodeEndNextShapeHelper({ relTypes: ['Connect'], data, prisma, userData })),
                }
            },
            update: async ({ data, prisma, userData }) => {
                return {
                    wasSuccessful: noNull(data.wasSuccessful),
                    ...(await nodeEndNextShapeHelper({ relTypes: ['Connect', 'Disconnect'], data, prisma, userData })),
                }
            },
        },
        yup: nodeEndValidation,
    }
})