import { Prisma } from "@prisma/client";
import { MaxObjects, NodeLoop, NodeLoopCreateInput, NodeLoopUpdateInput } from '@shared/consts';
import { nodeLoopValidation } from "@shared/validation";
import { noNull, shapeHelper } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { defaultPermissions } from "../utils";
import { NodeModel } from "./node";
import { ModelLogic } from "./types";

const __typename = 'NodeLoop' as const;
const suppFields = [] as const;
export const NodeLoopModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: NodeLoopCreateInput,
    GqlUpdate: NodeLoopUpdateInput,
    GqlModel: NodeLoop,
    GqlPermission: {},
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.node_loopUpsertArgs['create'],
    PrismaUpdate: Prisma.node_loopUpsertArgs['update'],
    PrismaModel: Prisma.node_loopGetPayload<SelectWrap<Prisma.node_loopSelect>>,
    PrismaSelect: Prisma.node_loopSelect,
    PrismaWhere: Prisma.node_loopWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.node_loop,
    // Doesn't make sense to have a displayer for this model
    display: {
        select: () => ({ id: true }),
        label: () => ''
    },
    format: {
        gqlRelMap: {
            __typename,
            whiles: 'NodeLoopWhile',
        },
        prismaRelMap: {
            __typename,
            node: 'Node',
            whiles: 'NodeLoopWhile',
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                loops: noNull(data.loops),
                maxLoops: noNull(data.maxLoops),
                operation: noNull(data.operation),
                ...(await shapeHelper({ relation: 'node', relTypes: ['Connect'], isOneToOne: true, isRequired: true, objectType: 'Node', parentRelationshipName: 'loop', data, ...rest })),
                ...(await shapeHelper({ relation: 'whiles', relTypes: ['Create'], isOneToOne: false, isRequired: false, objectType: 'NodeLoopWhile', parentRelationshipName: 'loop', data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                loops: noNull(data.loops),
                maxLoops: noNull(data.maxLoops),
                operation: noNull(data.operation),
                ...(await shapeHelper({ relation: 'node', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'Node', parentRelationshipName: 'loop', data, ...rest })),
                ...(await shapeHelper({ relation: 'whiles', relTypes: ['Create', 'Update', 'Delete'], isOneToOne: false, isRequired: false, objectType: 'NodeLoopWhile', parentRelationshipName: 'loop', data, ...rest })),
            })
        },
        yup: nodeLoopValidation,
    },
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ node: 'Node' }),
        permissionResolvers: defaultPermissions,
        owner: (data, userId) => NodeModel.validate!.owner(data.node as any, userId),
        isDeleted: (data, languages) => NodeModel.validate!.isDeleted(data.node as any, languages),
        isPublic: (data, languages) => NodeModel.validate!.isPublic(data.node as any, languages),
        visibility: {
            private: { node: NodeModel.validate!.visibility.private },
            public: { node: NodeModel.validate!.visibility.public },
            owner: (userId) => ({ node: NodeModel.validate!.visibility.owner(userId) }),
        }
    },
})