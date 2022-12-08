import { NodeLinkCreateInput, NodeLinkUpdateInput, NodeLinkWhenCreateInput, NodeLinkWhenTranslationCreateInput, NodeLinkWhenTranslationUpdateInput, NodeLinkWhenUpdateInput } from "graphql/generated/globalTypes";
import { NodeLink, NodeLinkWhen, NodeLinkWhenTranslation, ShapeWrapper } from "types";
import { hasObjectChanged } from "utils";
import { shapeCreateList, shapePrim, shapeUpdate, shapeUpdateList } from "./shapeTools";

export type NodeLinkWhenTranslationShape = Omit<ShapeWrapper<NodeLinkWhenTranslation>, 'language' | 'name'> & {
    id: string;
    language: NodeLinkWhenTranslationCreateInput['language'];
    name: NodeLinkWhenTranslationCreateInput['name'];
}

export type NodeLinkWhenShape = Omit<ShapeWrapper<NodeLinkWhen>, 'linkId'> & {
    id: string;
    linkId: NodeLinkWhenCreateInput['linkId'];
}

export const shapeNodeLinkWhenTranslationCreate = (item: NodeLinkWhenTranslationShape): NodeLinkWhenTranslationCreateInput => ({
    id: item.id,
    language: item.language,
    description: item.description,
    name: item.name,
})

export const shapeNodeLinkWhenTranslationUpdate = (
    original: NodeLinkWhenTranslationShape,
    updated: NodeLinkWhenTranslationShape
): NodeLinkWhenTranslationUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: u.id,
        ...shapePrim(o, u, 'description'),
        ...shapePrim(o, u, 'name'),
    }), 'id')

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
        ...shapePrim(o, u, 'linkId'),
        ...shapePrim(o, u, 'condition'),
        ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeNodeLinkWhenTranslationCreate, shapeNodeLinkWhenTranslationUpdate, 'id'),
    }), 'id')

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
        ...shapePrim(o, u, 'operation'),
        ...shapePrim(o, u, 'fromId'),
        ...shapePrim(o, u, 'toId'),
        ...shapeUpdateList({
            whens: o.whens?.map(w => ({ ...w, linkId: o.id })),
        }, {
            whens: u.whens?.map(w => ({ ...w, linkId: u.id })),
        }, 'whens', hasObjectChanged, shapeNodeLinkWhenCreate, shapeNodeLinkWhenUpdate, 'id'),
    }), 'id')