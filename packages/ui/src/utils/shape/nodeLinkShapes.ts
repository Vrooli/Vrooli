import { NodeLinkCreateInput, NodeLinkUpdateInput, NodeLinkWhenCreateInput, NodeLinkWhenTranslationCreateInput, NodeLinkWhenTranslationUpdateInput, NodeLinkWhenUpdateInput } from "graphql/generated/globalTypes";
import { NodeLink, NodeLinkWhen, NodeLinkWhenTranslation } from "types";
import { formatForUpdate, hasObjectChanged } from "utils";
import { shapeCreateList, shapeUpdateList, ShapeWrapper } from "./shapeTools";

type NodeLinkWhenTranslationCreate = ShapeWrapper<NodeLinkWhenTranslation> &
    Pick<NodeLinkWhenTranslationCreateInput, 'language' | 'title'>;
/**
 * Format a nodeLink's when's translations for create mutation.
 * @param translations Translations to format
 * @returns Formatted translations
 */
export const shapeNodeLinkWhenTranslationsCreate = (
    translations: NodeLinkWhenTranslationCreate[] | null | undefined
): NodeLinkWhenTranslationCreateInput[] | undefined => {
    return shapeCreateList(translations, (translation) => ({
        id: translation.id,
        language: translation.language,
        description: translation.description,
        title: translation.title,
    }))
}

interface NodeLinkWhenTranslationUpdate extends NodeLinkWhenTranslationCreate { id: string };
/**
 * Format a nodeLink's when's translations for update mutation.
 * @param original Original translations list
 * @param updated Updated translations list
 * @returns Formatted translations
 */
export const shapeNodeLinkWhenTranslationsUpdate = (
    original: NodeLinkWhenTranslationUpdate[] | null | undefined,
    updated: NodeLinkWhenTranslationUpdate[] | null | undefined
): {
    translationsCreate?: NodeLinkWhenTranslationCreateInput[],
    translationsUpdate?: NodeLinkWhenTranslationUpdateInput[],
    translationsDelete?: string[],
} => shapeUpdateList(
    original,
    updated,
    'translations',
    shapeNodeLinkWhenTranslationsCreate,
    hasObjectChanged,
    formatForUpdate as (original: NodeLinkWhenTranslationUpdate, updated: NodeLinkWhenTranslationUpdate) => NodeLinkWhenTranslationUpdateInput | undefined,
)

type NodeLinkWhenCreate = ShapeWrapper<NodeLinkWhen> & 
    Pick<NodeLinkWhenCreateInput, 'toId'>;
/**
 * Format a nodeLink when for create mutation.
 * @param when The nodeLink when's information
 * @returns NodeLink when shaped for create mutation
 */
export const shapeNodeLinkWhenCreate = (when: NodeLinkWhenCreate | null | undefined): NodeLinkWhenCreateInput | undefined => {
    if (!when) return undefined;
    return {
        id: when.id,
        toId: when.toId,
        condition: when.condition ?? '',
        ...shapeNodeLinkWhenTranslationsCreate(when.translations),
    };
}

interface NodeLinkWhenUpdate extends NodeLinkWhenCreate { id: string };
/**
 * Format a nodeLink when for update mutation
 * @param original The original nodeLink when's information
 * @param updated The updated nodeLink when's information
 * @returns NodeLink when shaped for update mutation
 */
export const shapeNodeLinkWhenUpdate = (original: NodeLinkWhenUpdate | null | undefined, updated: NodeLinkWhenUpdate | null | undefined): NodeLinkWhenUpdateInput | undefined => {
    if (!updated?.id) return undefined;
    if (!original) original = { id: updated.id } as NodeLinkWhenUpdate;
    return {
        id: original.id,
        toId: updated.toId !== original.toId ? updated.toId : undefined,
        condition: updated.condition !== original.condition ? updated.condition : undefined,
        ...shapeNodeLinkWhenTranslationsUpdate(original.translations, updated.translations),
    };
}

/**
 * Format an array of nodeLink whens for create mutation.
 * @param whens The nodeLink whens to format
 * @returns NodeLink whens shaped for create mutation
 */
export const shapeNodeLinkWhensCreate = (
    whens: NodeLinkWhenCreate[] | null | undefined
): NodeLinkWhenCreateInput[] | undefined => {
    return shapeCreateList(whens, shapeNodeLinkWhenCreate)
}

/**
 * Format an array of nodeLink whenss for update mutation.
 * @param original Original nodeLink whens list
 * @param updated Updated nodeLink whens list
 * @returns Formatted nodeLink whens
 */
export const shapeNodeLinkWhensUpdate = (
    original: NodeLinkWhenUpdate[] | null | undefined,
    updated: NodeLinkWhenUpdate[] | null | undefined
): {
    whensCreate?: NodeLinkWhenCreateInput[],
    whensUpdate?: NodeLinkWhenUpdateInput[],
    whensDelete?: string[],
} => shapeUpdateList(
    original,
    updated,
    'whens',
    shapeNodeLinkWhensCreate,
    hasObjectChanged,
    shapeNodeLinkWhenUpdate,
)

type NodeLinkCreate = ShapeWrapper<NodeLink> & 
    Pick<NodeLinkCreateInput, 'fromId' | 'toId'>;
/**
 * Format a nodeLink for create mutation.
 * @param nodeLink The nodeLink's information
 * @returns NodeLink shaped for create mutation
 */
export const shapeNodeLinkCreate = (nodeLink: NodeLinkCreate | null | undefined): NodeLinkCreateInput | undefined => {
    if (!nodeLink) return undefined;
    return {
        id: nodeLink.id,
        operation: nodeLink.operation,
        fromId: nodeLink.fromId,
        toId: nodeLink.toId,
        ...shapeNodeLinkWhensCreate(nodeLink.whens),
    };
}

interface NodeLinkUpdate extends NodeLinkCreate { id: string };
/**
 * Format a nodeLink for update mutation
 * @param original The original nodeLink's information
 * @param updated The updated nodeLink's information
 * @returns NodeLink shaped for update mutation
 */
export const shapeNodeLinkUpdate = (original: NodeLinkUpdate | null | undefined, updated: NodeLinkUpdate | null | undefined): NodeLinkUpdateInput | undefined => {
    if (!updated?.id) return undefined;
    if (!original) original = { id: updated.id } as NodeLinkUpdate;
    return {
        id: original.id,
        operation: updated.operation !== original.operation ? updated.operation : undefined,
        fromId: updated.fromId !== original.fromId ? updated.fromId : undefined,
        toId: updated.toId !== original.toId ? updated.toId : undefined,
        ...shapeNodeLinkWhensUpdate(original.whens, updated.whens),
    };
}

/**
 * Format an array nodeLinks for create mutation.
 * @param nodeLinks The nodeLinks to format
 * @returns NodeLinks shaped for create mutation
 */
export const shapeNodeLinksCreate = (
    nodeLinks: NodeLinkCreate[] | null | undefined
): NodeLinkCreateInput[] | undefined => {
    return shapeCreateList(nodeLinks, shapeNodeLinkCreate);
}

/**
 * Format an array of nodeLinks for update mutation.
 * @param original Original nodeLinks list
 * @param updated Updated nodeLinks list
 * @returns Formatted nodeLinks
 */
export const shapeNodeLinksUpdate = (
    original: NodeLinkUpdate[] | null | undefined,
    updated: NodeLinkUpdate[] | null | undefined
): {
    nodeLinksCreate?: NodeLinkCreateInput[],
    nodeLinksUpdate?: NodeLinkUpdateInput[],
    nodeLinksDelete?: string[],
} => shapeUpdateList(
    original,
    updated,
    'nodeLinks',
    shapeNodeLinksCreate,
    hasObjectChanged,
    shapeNodeLinkUpdate,
)