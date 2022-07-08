import { NodeLinkCreateInput, NodeLinkUpdateInput, NodeLinkWhenCreateInput, NodeLinkWhenTranslationCreateInput, NodeLinkWhenTranslationUpdateInput, NodeLinkWhenUpdateInput } from "graphql/generated/globalTypes";
import { NodeLink, NodeLinkWhen, NodeLinkWhenTranslation, ShapeWrapper } from "types";
import { hasObjectChanged } from "utils";
import { shapeCreateList, shapeUpdate, shapeUpdateList } from "./shapeTools";

export type NodeLinkWhenTranslationShape = Omit<ShapeWrapper<NodeLinkWhenTranslation>, 'language' | 'title'> & {
    id: string;
    language: NodeLinkWhenTranslationCreateInput['language'];
    title: NodeLinkWhenTranslationCreateInput['title'];
}

export const shapeNodeLinkWhenTranslationCreate = (item: NodeLinkWhenTranslationShape): NodeLinkWhenTranslationCreateInput => ({
    id: item.id,
    language: item.language,
    description: item.description,
    title: item.title,
})

export const shapeNodeLinkWhenTranslationUpdate = (
    original: NodeLinkWhenTranslationShape,
    updated: NodeLinkWhenTranslationShape
): NodeLinkWhenTranslationUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: u.id,
        description: u.description !== o.description ? u.description : undefined,
        title: u.title !== o.title ? u.title : undefined,
    }))

export const shapeNodeLinkWhenTranslationsCreate = (items: NodeLinkWhenTranslationShape[] | null | undefined): {
    translationsCreate?: NodeLinkWhenTranslationCreateInput[],
} => shapeCreateList(items, 'translations', shapeNodeLinkWhenTranslationCreate);

export const shapeNodeLinkWhenTranslationsUpdate = (
    o: NodeLinkWhenTranslationShape[] | null | undefined,
    u: NodeLinkWhenTranslationShape[] | null | undefined
): {
    translationsCreate?: NodeLinkWhenTranslationCreateInput[],
    translationsUpdate?: NodeLinkWhenTranslationUpdateInput[],
    translationsDelete?: string[],
} => shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeNodeLinkWhenTranslationCreate, shapeNodeLinkWhenTranslationUpdate)

export type NodeLinkWhenShape = Omit<ShapeWrapper<NodeLinkWhen>, 'linkId'> & {
    id: string;
    linkId: NodeLinkWhenCreateInput['linkId'];
}

export const shapeNodeLinkWhenCreate = (item: NodeLinkWhenShape): NodeLinkWhenCreateInput => ({
    id: item.id,
    linkId: item.linkId,
    condition: item.condition ?? '',
    ...shapeNodeLinkWhenTranslationsCreate(item.translations),
})

export const shapeNodeLinkWhenUpdate = (
    original: NodeLinkWhenShape,
    updated: NodeLinkWhenShape
): NodeLinkWhenUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: o.id,
        linkId: u.linkId !== o.linkId ? u.linkId : undefined,
        condition: u.condition !== o.condition ? u.condition : undefined,
        ...shapeNodeLinkWhenTranslationsUpdate(o.translations, u.translations),
    }))

export const shapeNodeLinkWhensCreate = (items: NodeLinkWhenShape[] | null | undefined): {
    whensCreate?: NodeLinkWhenCreateInput[],
} => shapeCreateList(items, 'whens', shapeNodeLinkWhenCreate);

export const shapeNodeLinkWhensUpdate = (
    o: NodeLinkWhenShape[] | null | undefined,
    u: NodeLinkWhenShape[] | null | undefined
): {
    whensCreate?: NodeLinkWhenCreateInput[],
    whensUpdate?: NodeLinkWhenUpdateInput[],
    whensDelete?: string[],
} => shapeUpdateList(o, u, 'whens', hasObjectChanged, shapeNodeLinkWhenCreate, shapeNodeLinkWhenUpdate)

export type NodeLinkShape = Omit<ShapeWrapper<NodeLink>, 'fromId' | 'toId'> & {
    id: string;
    fromId: NodeLinkCreateInput['fromId'];
    toId: NodeLinkCreateInput['toId'];
    whens?: Omit<NodeLinkWhenShape, 'linkId'>[];
}

export const shapeNodeLinkCreate = (item: NodeLinkShape): NodeLinkCreateInput => ({
    id: item.id,
    operation: item.operation,
    fromId: item.fromId,
    toId: item.toId,
    ...shapeNodeLinkWhensCreate(item.whens?.map(w => ({ ...w, linkId: item.id }))),
})

export const shapeNodeLinkUpdate = (
    original: NodeLinkShape,
    updated: NodeLinkShape
): NodeLinkUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: o.id,
        operation: u.operation !== o.operation ? u.operation : undefined,
        fromId: u.fromId !== o.fromId ? u.fromId : undefined,
        toId: u.toId !== o.toId ? u.toId : undefined,
        ...shapeNodeLinkWhensUpdate(o.whens?.map(w => ({ ...w, linkId: o.id })), u.whens?.map(w => ({ ...w, linkId: o.id }))),
    }))

export const shapeNodeLinksCreate = (items: NodeLinkShape[] | null | undefined): {
    nodeLinksCreate?: NodeLinkCreateInput[],
} => shapeCreateList(items, 'nodeLinks', shapeNodeLinkCreate);

export const shapeNodeLinksUpdate = (
    o: NodeLinkShape[] | null | undefined,
    u: NodeLinkShape[] | null | undefined
): {
    nodeLinksCreate?: NodeLinkCreateInput[],
    nodeLinksUpdate?: NodeLinkUpdateInput[],
    nodeLinksDelete?: string[],
} => shapeUpdateList(o, u, 'nodeLinks', hasObjectChanged, shapeNodeLinkCreate, shapeNodeLinkUpdate)