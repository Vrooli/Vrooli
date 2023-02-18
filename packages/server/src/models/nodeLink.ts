import { NodeLink, NodeLinkCreateInput, NodeLinkUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { Prisma } from "@prisma/client";
import { noNull, selPad, shapeHelper } from "../builders";
import { NodeModel } from "./node";
import { SelectWrap } from "../builders/types";
import { nodeLinkValidation } from '@shared/validation';

const __typename = 'NodeLink' as const;
const suppFields = [] as const;
export const NodeLinkModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: NodeLinkCreateInput,
    GqlUpdate: NodeLinkUpdateInput,
    GqlModel: NodeLink,
    GqlPermission: {},
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.node_linkUpsertArgs['create'],
    PrismaUpdate: Prisma.node_linkUpsertArgs['update'],
    PrismaModel: Prisma.node_linkGetPayload<SelectWrap<Prisma.node_linkSelect>>,
    PrismaSelect: Prisma.node_linkSelect,
    PrismaWhere: Prisma.node_linkWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.node_link,
    display: {
        select: () => ({
            id: true,
            from: selPad(NodeModel.display.select),
            to: selPad(NodeModel.display.select),
        }),
        // Label combines from and to labels
        label: (select, languages) => {
            return `${NodeModel.display.label(select.from as any, languages)} -> ${NodeModel.display.label(select.to as any, languages)}`
        }
    },
    format: {
        gqlRelMap: {
            __typename,
            whens: 'NodeLinkWhen',
        },
        prismaRelMap: {
            __typename,
            from: 'Node',
            to: 'Node',
            routineVersion: 'RoutineVersion',
            whens: 'NodeLinkWhen',
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ prisma, userData, data }) => ({
                id: data.id,
                operation: noNull(data.operation),
                ...(await shapeHelper({ relation: 'from', relTypes: ['Connect'], isOneToOne: true, isRequired: true, objectType: 'Node', parentRelationshipName: 'next', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'to', relTypes: ['Connect'], isOneToOne: true, isRequired: true, objectType: 'Node', parentRelationshipName: 'previous', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'routineVersion', relTypes: ['Connect'], isOneToOne: true, isRequired: true, objectType: 'RoutineVersion', parentRelationshipName: 'nodeLinks', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'whens', relTypes: ['Create'], isOneToOne: false, isRequired: false, objectType: 'NodeLinkWhen', parentRelationshipName: 'link', data, prisma, userData })),
    
            }),
            update: async ({ prisma, userData, data }) => ({
                operation: noNull(data.operation),
                ...(await shapeHelper({ relation: 'from', relTypes: ['Connect', 'Disconnect'], isOneToOne: true, isRequired: false, objectType: 'Node', parentRelationshipName: 'next', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'to', relTypes: ['Connect', 'Disconnect'], isOneToOne: true, isRequired: false, objectType: 'Node', parentRelationshipName: 'previous', data, prisma, userData })),
                ...(await shapeHelper({ relation: 'whens', relTypes: ['Create', 'Update', 'Delete'], isOneToOne: false, isRequired: false, objectType: 'NodeLinkWhen', parentRelationshipName: 'link', data, prisma, userData })),
            }),
        },
        yup: nodeLinkValidation,
    },
})