import { NodeLinkWhen, NodeLinkWhenCreateInput, NodeLinkWhenTranslation, NodeLinkWhenTranslationCreateInput, NodeLinkWhenTranslationUpdateInput, NodeLinkWhenUpdateInput } from "@shared/consts";
import { hasObjectChanged, createPrims, shapeUpdate, updatePrims, createRel, updateRel } from "utils";
import { ShapeModel } from "types";

export type NodeLinkWhenTranslationShape = Pick<NodeLinkWhenTranslation, 'id' | 'language' | 'description' | 'name'>

export type NodeLinkWhenShape = Pick<NodeLinkWhen, 'id' | 'condition'> & {
    link: { id: string };
    translations?: NodeLinkWhenTranslationShape[];
}

export const shapeNodeLinkWhenTranslation: ShapeModel<NodeLinkWhenTranslationShape, NodeLinkWhenTranslationCreateInput, NodeLinkWhenTranslationUpdateInput> = {
    create: (item) => createPrims(item, 'id', 'language', 'description', 'name'),
    update: (o, u) => shapeUpdate(u, updatePrims(o, u, 'id', 'description', 'name')),
}

export const shapeNodeLinkWhen: ShapeModel<NodeLinkWhenShape, NodeLinkWhenCreateInput, NodeLinkWhenUpdateInput> = {
    create: (item) => ({
        ...createPrims(item, 'id', 'condition'),
        ...createRel(item, 'link', ['Connect'], 'one'),
        ...createRel(item, 'translations', ['Create'], shapeNodeLinkWhenTranslation),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'condition'),
        ...updateRel(o, u, 'link', ['Connect'], 'one'),
        ...updateRel(o, u, 'translations', ['Create', 'Update', 'Delete'], hasObjectChanged, shapeNodeLinkWhenTranslation, 'id'),
    })
}