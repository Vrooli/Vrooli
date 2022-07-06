import { RoutineCreateInput, RoutineTranslationCreateInput, RoutineTranslationUpdateInput, RoutineUpdateInput } from "graphql/generated/globalTypes";
import { Routine, RoutineTranslation } from "types";
import { formatForUpdate, hasObjectChanged, ObjectType, shapeInputsCreate, shapeInputsUpdate, shapeNodeLinksCreate, shapeNodeLinksUpdate, shapeNodesCreate, shapeNodesUpdate, shapeOutputsCreate, shapeOutputsUpdate, shapeResourceListsCreate, shapeResourceListsUpdate, shapeTagsCreate, shapeTagsUpdate } from "utils";
import { shapeCreateList, shapeUpdate, shapeUpdateList, ShapeWrapper } from "./shapeTools";

type RoutineTranslationCreate = ShapeWrapper<RoutineTranslation> &
    Pick<RoutineTranslation, 'language' | 'instructions' | 'title'>;
/**
 * Format a routine's translations for create mutation.
 * @param translations Translations to format
 * @returns Formatted translations
 */
export const shapeRoutineTranslationsCreate = (
    translations: RoutineTranslationCreate[] | null | undefined
): RoutineTranslationCreateInput[] | undefined => {
    return shapeCreateList(translations, (translation) => ({
        id: translation.id,
        language: translation.language,
        description: translation.description,
        instructions: translation.instructions,
        title: translation.title,
    }))
}

interface RoutineTranslationUpdate extends RoutineTranslationCreate { id: string };
/**
 * Format a routine's translations for update mutation.
 * @param original Original translations list
 * @param updated Updated translations list
 * @returns Formatted translations
 */
export const shapeRoutineTranslationsUpdate = (
    original: RoutineTranslationUpdate[] | null | undefined,
    updated: RoutineTranslationUpdate[] | null | undefined
): {
    translationsCreate?: RoutineTranslationCreateInput[],
    translationsUpdate?: RoutineTranslationUpdateInput[],
    translationsDelete?: string[],
} => shapeUpdateList(
    original,
    updated,
    'translations',
    shapeRoutineTranslationsCreate,
    hasObjectChanged,
    formatForUpdate as (original: RoutineTranslationUpdate, updated: RoutineTranslationUpdate) => RoutineTranslationUpdateInput | undefined,
)

type RoutineCreate = ShapeWrapper<Routine>;
/**
 * Format a routine for create mutation.
 * @param routine The routine's information
 * @returns Routine shaped for create mutation
 */
export const shapeRoutineCreate = (routine: RoutineCreate | null | undefined): RoutineCreateInput | undefined => {
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

interface RoutineUpdate extends RoutineCreate { id: string };
/**
 * Format a routine for update mutation
 * @param original The original routine's information
 * @param updated The updated routine's information
 * @returns Routine shaped for update mutation
 */
export const shapeRoutineUpdate = (
    original: RoutineUpdate,
    updated: RoutineUpdate | null | undefined
): RoutineUpdateInput | undefined => shapeUpdate(original, updated, (o, u) => ({
    id: o.id,
    isAutomatable: u.isAutomatable,
    isComplete: u.isComplete,
    isInternal: u.isInternal,
    version: u.version,
    parentId: u.parent?.id,
    // projectId: u.p
    userId: u.owner?.__typename === ObjectType.User ? u.owner.id : undefined,
    organizationId: u.owner?.__typename === ObjectType.Organization ? u.owner.id : undefined,
    ...shapeNodesUpdate(o.nodes, u.nodes),
    ...shapeNodeLinksUpdate(o.nodeLinks, u.nodeLinks),
    ...shapeInputsUpdate(o.inputs, u.inputs),
    ...shapeOutputsUpdate(o.outputs, u.outputs),
    ...shapeRoutineTranslationsUpdate(o.translations, u.translations),
    ...shapeResourceListsUpdate(o.resourceLists, u.resourceLists),
    ...shapeTagsUpdate(o.tags ?? [], u.tags ?? []),
}))

/**
 * Format an array of routines for create mutation.
 * @param routines The routines to format
 * @returns Routines shaped for create mutation
 */
export const shapeRoutinesCreate = (
    routines: RoutineCreate[] | null | undefined
): RoutineCreateInput[] | undefined => {
    return shapeCreateList(routines, shapeRoutineCreate)
}

/**
 * Format an array of routines for update mutation.
 * @param original Original routines list
 * @param updated Updated routines list
 * @returns Formatted routines
 */
export const shapeRoutinesUpdate = (
    original: RoutineUpdate[] | null | undefined,
    updated: RoutineUpdate[] | null | undefined
): {
    routinesCreate?: RoutineCreateInput[],
    routinesUpdate?: RoutineUpdateInput[],
    routinesDelete?: string[],
} => shapeUpdateList(
    original,
    updated,
    'routines',
    shapeRoutinesCreate,
    hasObjectChanged,
    shapeRoutineUpdate,
)