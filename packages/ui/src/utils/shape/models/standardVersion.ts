import { StandardVersion, StandardVersionCreateInput, StandardVersionTranslation, StandardVersionUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { createPrims, shapeUpdate, updatePrims } from "./tools";

export type StandardVersionTranslationShape = Pick<StandardVersionTranslation, 'id' | 'language' | 'description' | 'jsonVariable'>

export type StandardVersionShape = Pick<StandardVersion, 'id'>

export const shapeStandardVersionTranslation: ShapeModel<StandardVersionTranslationShape, StandardVersionTranslationCreateInput, StandardVersionTranslationUpdateInput> = {
    create: (item) => createPrims(item, 'id', 'language', 'description', 'jsonVariable'),
    update: (o, u) => shapeUpdate(u, updatePrims(o, u, 'id', 'description', 'jsonVariable'))
}

export const shapeStandardVersion: ShapeModel<StandardVersionShape, StandardVersionCreateInput, StandardVersionUpdateInput> = {
    create: (item) => ({}) as any,
    update: (o, u) => ({}) as any
}