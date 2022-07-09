import { NodeLinkCreateInput, NodeLinkUpdateInput, NodeLinkWhenCreateInput, NodeLinkWhenTranslationCreateInput, NodeLinkWhenTranslationUpdateInput, NodeLinkWhenUpdateInput } from "graphql/generated/globalTypes";
import { NodeLink, NodeLinkWhen, NodeLinkWhenTranslation, ShapeWrapper } from "types";
import { hasObjectChanged } from "utils";
import { shapeCreateList, shapeUpdate, shapeUpdateList } from "./shapeTools";

export type NodeLinkWhenTranslationShape = Omit<ShapeWrapper<NodeLinkWhenTranslation>, 'language' | 'title'> & {
    id: string;
    language: NodeLinkWhenTranslationCreateInput['language'];
    title: NodeLinkWhenTranslationCreateInput['title'];
}

export type NodeLinkWhenShape = Omit<ShapeWrapper<NodeLinkWhen>, 'linkId'> & {
    id: string;
    linkId: NodeLinkWhenCreateInput['linkId'];
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

export const shapeNodeLinkWhenCreate = (item: NodeLinkWhenShape): NodeLinkWhenCreateInput => ({
    id: item.id,
    linkId: item.linkId,
    condition: item.condition ?? '',
    ...shapeCreateList(item, 'translations', shapeNodeLinkWhenTranslationCreate),
})

export const shapeNodeLinkWhenUpdate = (
    original: NodeLinkWhenShape,
    updated: NodeLinkWhenShape
): NodeLinkWhenUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: o.id,
        linkId: u.linkId !== o.linkId ? u.linkId : undefined,
        condition: u.condition !== o.condition ? u.condition : undefined,
        ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeNodeLinkWhenTranslationCreate, shapeNodeLinkWhenTranslationUpdate),
    }))

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
    ...shapeCreateList({ whens: item.whens?.map(w => ({ ...w, linkId: item.id })) }, 'whens', shapeNodeLinkWhenCreate),
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
        ...shapeUpdateList({
            whens: o.whens?.map(w => ({ ...w, linkId: o.id })),
        }, {
            whens: u.whens?.map(w => ({ ...w, linkId: u.id })),
        }, 'whens', hasObjectChanged, shapeNodeLinkWhenCreate, shapeNodeLinkWhenUpdate),
    }))