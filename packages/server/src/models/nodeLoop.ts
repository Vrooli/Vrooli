import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { NodeLoop, NodeLoopCreateInput, NodeLoopUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { defaultPermissions } from "../utils";
import { NodeModel } from "./node";

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
    mutate: {} as any,
    validate: {
        isTransferable: false,
        maxObjects: 100000,
        permissionsSelect: () => ({ node: 'Node' }),
        permissionResolvers: defaultPermissions,
        owner: (data) => NodeModel.validate!.owner(data.node as any),
        isDeleted: (data, languages) => NodeModel.validate!.isDeleted(data.node as any, languages),
        isPublic: (data, languages) => NodeModel.validate!.isPublic(data.node as any, languages),
        visibility: {
            private: { node: NodeModel.validate!.visibility.private },
            public: { node: NodeModel.validate!.visibility.public },
            owner: (userId) => ({ node: NodeModel.validate!.visibility.owner(userId) }),
        }
    },
})