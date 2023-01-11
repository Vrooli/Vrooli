import { StandardVersion, StandardVersionCreateInput, StandardVersionTranslation, StandardVersionTranslationCreateInput, StandardVersionTranslationUpdateInput, StandardVersionUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { ResourceListShape } from "./resourceList";
import { StandardShape } from "./standard";
import { createPrims, shapeUpdate, updatePrims } from "./tools";

export type StandardVersionTranslationShape = Pick<StandardVersionTranslation, 'id' | 'language' | 'description' | 'jsonVariable'>

export type StandardVersionShape = Pick<StandardVersion, 'id' | 'isComplete' | 'isLatest' | 'isPrivate' | 'isFile' | 'default' | 'props' | 'yup' | 'standardType' | 'versionIndex' | 'versionLabel' | 'versionNotes'> & {
    directoryListings?: { id: string }[];
    root: StandardShape;
    resourceList?: ResourceListShape;
    translations?: StandardVersionTranslationShape[];
}

export const shapeStandardVersionTranslation: ShapeModel<StandardVersionTranslationShape, StandardVersionTranslationCreateInput, StandardVersionTranslationUpdateInput> = {
    create: (d) => createPrims(d, 'id', 'language', 'description', 'jsonVariable'),
    update: (o, u) => shapeUpdate(u, updatePrims(o, u, 'id', 'description', 'jsonVariable'))
}

export const shapeStandardVersion: ShapeModel<StandardVersionShape, StandardVersionCreateInput, StandardVersionUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u) => shapeUpdate(u, {}) as any
}