import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { MaxObjects, NodeLoopWhile, NodeLoopWhileCreateInput, NodeLoopWhileUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { defaultPermissions } from "../utils";
import { NodeLoopModel } from "./nodeLoop";
import { nodeLoopWhileValidation } from "@shared/validation";

const __typename = 'NodeLoopWhile' as const;

const suppFields = [] as const;
export const NodeLoopWhileModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: NodeLoopWhileCreateInput,
    GqlUpdate: NodeLoopWhileUpdateInput,
    GqlModel: NodeLoopWhile,
    GqlPermission: {},
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.node_loop_whileUpsertArgs['create'],
    PrismaUpdate: Prisma.node_loop_whileUpsertArgs['update'],
    PrismaModel: Prisma.node_loop_whileGetPayload<SelectWrap<Prisma.node_loop_whileSelect>>,
    PrismaSelect: Prisma.node_loop_whileSelect,
    PrismaWhere: Prisma.node_loop_whileWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.node_loop_while,
    // Doesn't make sense to have a displayer for this model
    display: {
        select: () => ({ id: true }),
        label: () => ''
    },
    format: {
        gqlRelMap: {
            __typename,
        },
        prismaRelMap: {
            __typename,
            loop: 'NodeLoop',
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, prisma, userData }) => ({
                id: data.id,
                //TODO
            } as any),
            update: async ({ data, prisma, userData }) => ({
                id: data.id,
                //TODO
            } as any)
        },
        yup: nodeLoopWhileValidation,
    },
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({ loop: 'NodeLoop' }),
        permissionResolvers: defaultPermissions,
        owner: (data) => NodeLoopModel.validate!.owner(data.loop as any),
        isDeleted: (data, languages) => NodeLoopModel.validate!.isDeleted(data.loop as any, languages),
        isPublic: (data, languages) => NodeLoopModel.validate!.isPublic(data.loop as any, languages),
        visibility: {
            private: { loop: NodeLoopModel.validate!.visibility.private },
            public: { loop: NodeLoopModel.validate!.visibility.public },
            owner: (userId) => ({ loop: NodeLoopModel.validate!.visibility.owner(userId) }),
        }
    },
})