import { NodeLinkWhen, NodeLinkWhenCreateInput, NodeLinkWhenTranslation, NodeLinkWhenTranslationCreateInput, NodeLinkWhenTranslationUpdateInput, NodeLinkWhenUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";

export type NodeLinkWhenTranslationShape = Pick<NodeLinkWhenTranslation, 'id' | 'language' | 'description' | 'name'> & {
    __typename?: 'NodeLinkWhenTranslation';
}

export type NodeLinkWhenShape = Pick<NodeLinkWhen, 'id' | 'condition'> & {
    __typename?: 'NodeLinkWhen';
    link: { id: string };
    translations?: NodeLinkWhenTranslationShape[] | null;
}

export const shapeNodeLinkWhenTranslation: ShapeModel<NodeLinkWhenTranslationShape, NodeLinkWhenTranslationCreateInput, NodeLinkWhenTranslationUpdateInput> = {
    create: (d) => createPrims(d, 'id', 'language', 'description', 'name'),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, 'id', 'description', 'name'), a),
}

export const shapeNodeLinkWhen: ShapeModel<NodeLinkWhenShape, NodeLinkWhenCreateInput, NodeLinkWhenUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'condition'),
        ...createRel(d, 'link', ['Connect'], 'one'),
        ...createRel(d, 'translations', ['Create'], 'many', shapeNodeLinkWhenTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'condition'),
        ...updateRel(o, u, 'link', ['Connect'], 'one'),
        ...updateRel(o, u, 'translations', ['Create', 'Update', 'Delete'], 'many', shapeNodeLinkWhenTranslation),
    }, a)
}