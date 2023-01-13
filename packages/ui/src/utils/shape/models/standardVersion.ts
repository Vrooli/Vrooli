import { StandardVersion, StandardVersionCreateInput, StandardVersionTranslation, StandardVersionTranslationCreateInput, StandardVersionTranslationUpdateInput, StandardVersionUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { ResourceListShape } from "./resourceList";
import { StandardShape } from "./standard";
import { createPrims, shapeUpdate, updatePrims } from "./tools";

export type StandardVersionTranslationShape = Pick<StandardVersionTranslation, 'id' | 'language' | 'description' | 'jsonVariable'>

export type StandardVersionShape = Pick<StandardVersion, 'id' | 'isComplete' | 'isLatest' | 'isPrivate' | 'isFile' | 'default' | 'props' | 'yup' | 'standardType' | 'versionIndex' | 'versionLabel' | 'versionNotes'> & {
    directoryListings?: { id: string }[] | null;
    root: StandardShape;
    resourceList?: ResourceListShape | null;
    translations?: StandardVersionTranslationShape[] | null;
}

export const shapeStandardVersionTranslation: ShapeModel<StandardVersionTranslationShape, StandardVersionTranslationCreateInput, StandardVersionTranslationUpdateInput> = {
    create: (d) => createPrims(d, 'id', 'language', 'description', 'jsonVariable'),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, 'id', 'description', 'jsonVariable'), a)
}

export const shapeStandardVersion: ShapeModel<StandardVersionShape, StandardVersionCreateInput, StandardVersionUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u, a) => shapeUpdate(u, {}, a) as any
}