import { NodeRoutineList, NodeRoutineListCreateInput, NodeRoutineListUpdateInput } from "../schema/types";
import { relationshipToPrisma, RelationshipTypes } from "./builder";
import { nodeRoutineListCreate, nodeRoutineListItemsCreate, nodeRoutineListItemsUpdate, nodeRoutineListUpdate, nodeRoutineListItemTranslationCreate, nodeRoutineListItemTranslationUpdate } from "@shared/validation";
import { PrismaType } from "../types";
import { RoutineModel } from "./routine";
import { TranslationModel } from "./translation";
import { FormatConverter, GraphQLModelType } from "./types";

//==============================================================
/* #region Custom Components */
//==============================================================

export const nodeRoutineListFormatter = (): FormatConverter<NodeRoutineList, any> => ({
    relationshipMap: {
        '__typename': 'NodeRoutineList',
        'routines': {
            '__typename': 'NodeRoutineListItem',
            'routine': 'Routine',
        },
    },
})

export const nodeRoutineListMutater = (prisma: PrismaType) => ({
    async toDBShapeCreate(userId: string | null, data: NodeRoutineListCreateInput): Promise<any> {
        return {
            id: data.id,
            isOrdered: data.isOrdered,
            isOptional: data.isOptional,
            routines: await this.relationshipBuilderRoutineListNodeItem(userId, data, true)
        }
    },
    async toDBShapeUpdate(userId: string | null, data: NodeRoutineListUpdateInput): Promise<any> {
        return {
            isOrdered: data.isOrdered,
            isOptional: data.isOptional,
            routines: await this.relationshipBuilderRoutineListNodeItem(userId, data, false)
        }
    },
    /**
     * Add, update, or remove routine list node item data from a node
     */
    async relationshipBuilderRoutineListNodeItem(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): Promise<{ [x: string]: any } | undefined> {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by node data (since they can only be applied to one node)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName: 'routines', isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        const mutate = RoutineModel.mutate(prisma);
        // Validate create
        if (Array.isArray(formattedInput.create)) {
            // Check for valid arguments
            nodeRoutineListItemsCreate.validateSync(formattedInput.create, { abortEarly: false });
            let result: { [x: string]: any }[] = [];
            for (const data of formattedInput.create) {
                result.push({
                    id: data.id,
                    index: data.index,
                    isOptional: data.isOptional,
                    routineId: await mutate.relationshipBuilder(userId, data, isAdd),
                    translations: TranslationModel.relationshipBuilder(userId, data, { create: nodeRoutineListItemTranslationCreate, update: nodeRoutineListItemTranslationUpdate }, false),
                })
            }
            formattedInput.create = result;
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            // Check for valid arguments
            nodeRoutineListItemsUpdate.validateSync(formattedInput.update.map(u => u.data), { abortEarly: false });
            let result: { where: { [x: string]: string }, data: { [x: string]: any } }[]  = [];
            for (const data of formattedInput.update) {
                result.push({
                    where: data.where,
                    data: {
                        index: data.data.index ?? undefined,
                        isOptional: data.data.isOptional ?? undefined,
                        routineId: await mutate.relationshipBuilder(userId, data.data, isAdd),
                        translations: TranslationModel.relationshipBuilder(userId, data.data, { create: nodeRoutineListItemTranslationCreate, update: nodeRoutineListItemTranslationUpdate }, false),
                    }
                })
            }
            formattedInput.update = result;
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    /**
     * Add, update, or remove routine list node data from a node.
     * Since this is a one-to-one relationship, we cannot return arrays
     */
    async relationshipBuilder(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): Promise<{ [x: string]: any } | undefined> {
        // Convert input to Prisma shape
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by node data (since they can only be applied to one node)
        let formattedInput: any = relationshipToPrisma({ data: input, relationshipName: 'nodeRoutineList', isAdd, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        // Validate create
        if (Array.isArray(formattedInput.create) && formattedInput.create.length > 0) {
            const create = formattedInput.create[0];
            // Check for valid arguments
            nodeRoutineListCreate.validateSync(create, { abortEarly: false });
            // Convert nested relationships
            formattedInput.create = await this.toDBShapeCreate(userId, create);
        }
        // Validate update
        if (Array.isArray(formattedInput.update) && formattedInput.update.length > 0) {
            const update = formattedInput.update[0].data;
            // Check for valid arguments
            nodeRoutineListUpdate.validateSync(update, { abortEarly: false });
            // Convert nested relationships
            formattedInput.update = await this.toDBShapeUpdate(userId, update);
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export const NodeRoutineListModel = ({
    prismaObject: (prisma: PrismaType) => prisma.node_routine_list,
    format: nodeRoutineListFormatter(),
    mutate: nodeRoutineListMutater,
    type: 'NodeRoutineList' as GraphQLModelType,
})

//==============================================================
/* #endregion Model */
//==============================================================