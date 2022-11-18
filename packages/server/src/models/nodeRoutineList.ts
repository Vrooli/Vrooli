import { NodeRoutineList, NodeRoutineListCreateInput, NodeRoutineListUpdateInput } from "../schema/types";
import { relationshipBuilderHelper } from "./builder";
import { nodeRoutineListItemTranslationCreate, nodeRoutineListItemTranslationUpdate } from "@shared/validation";
import { PrismaType } from "../types";
import { RoutineModel } from "./routine";
import { TranslationModel } from "./translation";
import { FormatConverter, GraphQLModelType, Mutater } from "./types";
import { Prisma } from "@prisma/client";

export const nodeRoutineListFormatter = (): FormatConverter<NodeRoutineList, any> => ({
    relationshipMap: {
        __typename: 'NodeRoutineList',
        routines: {
            __typename: 'NodeRoutineListItem',
            routine: 'Routine',
        },
    },
})

export const nodeRoutineListMutater = (prisma: PrismaType): Mutater<NodeRoutineList> => ({
    async shapeCreate(userId: string, data: NodeRoutineListCreateInput): Promise<Prisma.node_routine_listCreateNestedOneWithoutNodeInput['create']> {
        return {
            id: data.id,
            isOrdered: data.isOrdered ?? undefined,
            isOptional: data.isOptional ?? undefined,
            routines: await this.relationshipBuilderRoutineListNodeItem(userId, data, true)
        }
    },
    async shapeUpdate(userId: string, data: NodeRoutineListUpdateInput): Promise<Prisma.node_routine_listUpdateOneWithoutNodeNestedInput['update']> {
        return {
            isOrdered: data.isOrdered ?? undefined,
            isOptional: data.isOptional ?? undefined,
            routines: await this.relationshipBuilderRoutineListNodeItem(userId, data, false)
        }
    },
    /**
     * Add, update, or remove routine list node item data from a node
     */
    async relationshipBuilderRoutineListNodeItem(
        userId: string,
        data: { [x: string]: any },
        isAdd: boolean = true,
    ): Promise<{ [x: string]: any } | undefined> {
        return relationshipBuilderHelper({
            data,
            relationshipName: 'routine',
            isAdd,
            isTransferable: false,
            shape: {
                shapeCreate: async (userId, cuData) => ({
                    id: cuData.id,
                    index: cuData.index,
                    isOptional: cuData.isOptional,
                    routineId: await RoutineModel.mutate(prisma).relationshipBuilder!(userId, cuData, true),
                    translations: TranslationModel.relationshipBuilder(userId, cuData, { create: nodeRoutineListItemTranslationCreate, update: nodeRoutineListItemTranslationUpdate }, true),
                }),
                shapeUpdate: async (userId, cuData) => ({
                    index: cuData.index ?? undefined,
                    isOptional: cuData.isOptional ?? undefined,
                    routineId: await RoutineModel.mutate(prisma).relationshipBuilder!(userId, cuData, false),
                    translations: TranslationModel.relationshipBuilder(userId, cuData, { create: nodeRoutineListItemTranslationCreate, update: nodeRoutineListItemTranslationUpdate }, false),
                }),
            },
            userId,
        })
    },
    /**
     * Add, update, or remove routine list node data from a node.
     * Since this is a one-to-one relationship, we cannot return arrays
     */
    async relationshipBuilder(
        userId: string,
        data: { [x: string]: any },
        isAdd: boolean = true,
    ): Promise<{ [x: string]: any } | undefined> {
        return relationshipBuilderHelper({
            data,
            relationshipName: 'nodeRoutineList',
            isAdd,
            isTransferable: false,
            shape: { shapeCreate: this.shapeCreate, shapeUpdate: this.shapeUpdate },
            userId,
        });
    },
})

export const NodeRoutineListModel = ({
    prismaObject: (prisma: PrismaType) => prisma.node_routine_list,
    format: nodeRoutineListFormatter(),
    mutate: nodeRoutineListMutater,
    type: 'NodeRoutineList' as GraphQLModelType,
})