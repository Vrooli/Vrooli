import { NodeLink, NodeLinkCreateInput, NodeLinkUpdateInput, NodeLinkWhen, NodeLinkWhenCreateInput, NodeLinkWhenTranslation, NodeLinkWhenTranslationCreateInput, NodeLinkWhenUpdateInput } from "@shared/consts";
import { OmitCalculated } from "types";
import { hasObjectChanged } from "utils";
import { shapeCreateList, shapeCreatePrims, shapeUpdatePrims, shapeUpdate, shapeUpdateList } from "./shapeTools";

export type NodeLinkWhenTranslationShape = OmitCalculated<NodeLinkWhenTranslation>

export type NodeLinkWhenShape = Omit<OmitCalculated<NodeLinkWhen>, 'linkId'> & {
    linkId: NodeLinkWhenCreateInput['linkId'];
}

export const shapeNodeLinkWhenTranslationCreate = (item: NodeLinkWhenTranslationShape): NodeLinkWhenTranslationCreateInput =>
    shapeCreatePrims(item, 'id', 'language', 'description', 'name')

export const shapeNodeLinkWhenTranslationUpdate = (o: NodeLinkWhenTranslationShape, u: NodeLinkWhenTranslationShape): NodeLinkWhenTranslationUpdateInput | undefined =>
    shapeUpdate(u, shapeUpdatePrims(o, u, 'id', 'description', 'name'))

export const shapeNodeLinkWhenCreate = (item: NodeLinkWhenShape): NodeLinkWhenCreateInput => ({
    id: item.id,
    linkId: item.linkId,
    condition: item.condition ?? '',
    ...shapeCreateList(item, 'translations', shapeNodeLinkWhenTranslationCreate),
})

export const shapeNodeLinkWhenUpdate = (o: NodeLinkWhenShape, u: NodeLinkWhenShape): NodeLinkWhenUpdateInput | undefined =>
    shapeUpdate(u, {
        ...shapeUpdatePrims(o, u, 'id', 'linkId', 'condition'),
        ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeNodeLinkWhenTranslationCreate, shapeNodeLinkWhenTranslationUpdate, 'id'),
    })

export type NodeLinkShape = Omit<OmitCalculated<NodeLink>, 'fromId' | 'toId'> & {
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

export const shapeNodeLinkUpdate = (o: NodeLinkShape, u: NodeLinkShape): NodeLinkUpdateInput | undefined =>
    shapeUpdate(u, {
        ...shapeUpdatePrims(o, u, 'id', 'operation', 'fromId', 'toId'),
        ...shapeUpdateList({
            whens: o.whens?.map(w => ({ ...w, linkId: o.id })),
        }, {
            whens: u.whens?.map(w => ({ ...w, linkId: u.id })),
        }, 'whens', hasObjectChanged, shapeNodeLinkWhenCreate, shapeNodeLinkWhenUpdate, 'id'),
    })