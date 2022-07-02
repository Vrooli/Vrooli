import { RoutineCreateInput, RoutineUpdateInput } from "graphql/generated/globalTypes";
import { Routine } from "types";
import { formatForUpdate, hasObjectChanged, ObjectType, shapeInputsCreate, shapeOutputsCreate, shapeResourceListsCreate, shapeResourceListsUpdate, shapeTagsCreate, shapeTagsUpdate } from "utils";

/**
 * Format a routine's translations for create mutation.
 * @param translations Translations to format
 * @returns Formatted translations
 */
export const shapeRoutineTranslationsCreate = (translations: { [x: string]: any }[] | null | undefined): { translationsCreate?: RoutineCreateInput['translationsCreate'] } => {
    if (!translations) return {};
    const formatted: RoutineCreateInput['translationsCreate'] = [];
    for (const translation of translations) {
        formatted.push({
            id: translation.id,
            language: translation.language,
            description: translation.description,
            instructions: translation.instructions,
            title: translation.title,
        });
    }
    return formatted.length > 0 ? {
        translationsCreate: formatted,
    } : {};
}

/**
 * Format a routine's translations for update mutation.
 * @param original Original translations list
 * @param updated Updated translations list
 * @returns Formatted translations
 */
export const shapeRoutineTranslationsUpdate = (
    original: { [x: string]: any }[] | null | undefined,
    updated: { [x: string]: any }[] | null | undefined
): {
    translationsCreate?: RoutineUpdateInput['translationsCreate'],
    translationsUpdate?: RoutineUpdateInput['translationsUpdate'],
    translationsDelete?: RoutineUpdateInput['translationsDelete'],
} => {
    if (!updated) return {};
    if (!original || !Array.isArray(original)) {
        return shapeRoutineTranslationsCreate(updated);
    }
    return Array.isArray(updated) && updated.length > 0 ? {
        ...(shapeRoutineTranslationsCreate(updated.filter(t => !original.some(o => o.id === t.id)))),
        translationsUpdate: updated.map(t => {
            const ot = original.find(o => o.id === t.id);
            return (ot && hasObjectChanged(ot, t)) ? formatForUpdate(ot, t) : undefined;
        }).filter(t => Boolean(t)) as RoutineUpdateInput['translationsUpdate'],
        translationsDelete: original.filter(o => !updated.some(u => u.id === o.id)).map(o => o.id),
    } : {}
}

type ShapeRoutineCreateInput = Partial<Routine> & {

}
/**
 * Format a routine for create mutation.
 * @param routine The routine's information
 * @returns Routine shaped for create mutation
 */
export const shapeRoutineCreate = (routine: ShapeRoutineCreateInput | null | undefined): RoutineCreateInput | undefined => {
    if (!routine) return undefined;
    return {
        id: routine.id,
        isAutomatable: routine.isAutomatable,
        isComplete: routine.isComplete,
        isInternal: routine.isInternal,
        version: routine.version,
        parentId: routine.parent?.id,
        // projectId: routine.p
        createdByUserId: routine.owner?.__typename === ObjectType.User ? routine.owner.id : undefined,
        createdByOrganizationId: routine.owner?.__typename === ObjectType.Organization ? routine.owner.id : undefined,
        ...shapeNodesCreate(routine.nodes),
        ...shapeNodeLinksCreate(routine.nodeLinks),
        ...shapeInputsCreate(routine.inputs),
        ...shapeOutputsCreate(routine.outputs),
        ...shapeRoutineTranslationsCreate(routine.translations),
        ...shapeResourceListsCreate(routine.resourceLists),
        ...shapeTagsCreate(routine.tags ?? []),
    };
}

type ShapeRoutineUpdateInput = Partial<Routine> & {
    id: Routine['id'],
};
/**
 * Format a routine for update mutation
 * @param original The original routine's information
 * @param updated The updated routine's information
 * @returns Routine shaped for update mutation
 */
export const shapeRoutineUpdate = (original: ShapeRoutineUpdateInput | null | undefined, updated: ShapeRoutineUpdateInput | null | undefined): RoutineUpdateInput | undefined => {
    if (!updated?.id) return undefined;
    if (!original) original = { id: updated.id };
    return {
        id: original.id,
        isAutomatable: updated.isAutomatable,
        isComplete: updated.isComplete,
        isInternal: updated.isInternal,
        version: updated.version,
        parentId: updated.parent?.id,
        // projectId: updated.p
        createdByUserId: updated.owner?.__typename === ObjectType.User ? updated.owner.id : undefined,
        createdByOrganizationId: updated.owner?.__typename === ObjectType.Organization ? updated.owner.id : undefined,
        ...shapeNodesUpdate(original.nodes, updated.nodes),
        ...shapeNodeLinksUpdate(original.nodeLinks, updated.nodeLinks),
        ...shapeInputsUpdate(original.inputs, updated.inputs),
        ...shapeOutputsUpdate(original.outputs, updated.outputs),
        ...shapeRoutineTranslationsUpdate(original.translations, updated.translations),
        ...shapeResourceListsUpdate(original.resourceLists, updated.resourceLists),
        ...shapeTagsUpdate(original.tags ?? [], updated.tags ?? []),
    };
}