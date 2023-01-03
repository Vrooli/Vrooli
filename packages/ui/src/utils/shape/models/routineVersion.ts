import { RoutineVersion, RoutineVersionCreateInput, RoutineVersionTranslation, RoutineVersionTranslationCreateInput, RoutineVersionTranslationUpdateInput, RoutineVersionUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { createPrims, shapeUpdate, updatePrims } from "./tools";

export type RoutineVersionTranslationShape = Pick<RoutineVersionTranslation, 'id' | 'language' | 'description' | 'instructions' | 'name'>

export type RoutineVersionShape = Pick<RoutineVersion, 'id'>

export const shapeRoutineVersionTranslation: ShapeModel<RoutineVersionTranslationShape, RoutineVersionTranslationCreateInput, RoutineVersionTranslationUpdateInput> = {
    create: (item) => createPrims(item, 'id', 'language', 'description', 'instructions', 'name'),
    update: (o, u) => shapeUpdate(u, updatePrims(o, u, 'id', 'description', 'instructions', 'name'))
}

export const shapeRoutineVersion: ShapeModel<RoutineVersionShape, RoutineVersionCreateInput, RoutineVersionUpdateInput> = {
    create: (item) => ({}) as any,
    update: (o, u) => ({}) as any
}