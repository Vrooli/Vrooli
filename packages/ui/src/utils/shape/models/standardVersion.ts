import { StandardVersion, StandardVersionCreateInput, StandardVersionTranslation, StandardVersionTranslationCreateInput, StandardVersionTranslationUpdateInput, StandardVersionUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { ResourceListShape } from "./resourceList";
import { StandardShape } from "./standard";
import { createPrims, shapeUpdate, updatePrims } from "./tools";

export type StandardVersionTranslationShape = Pick<StandardVersionTranslation, 'id' | 'language' | 'description' | 'jsonVariable' | 'name'> & {
    __typename?: 'StandardVersionTranslation';
}

export type StandardVersionShape = Pick<StandardVersion, 'id' | 'isComplete' | 'isPrivate' | 'isFile' | 'default' | 'props' | 'yup' | 'standardType' | 'versionLabel' | 'versionNotes'> & {
    __typename?: 'StandardVersion';
    directoryListings?: { id: string }[] | null;
    root: StandardShape;
    resourceList?: { id: string } | ResourceListShape | null;
    translations?: StandardVersionTranslationShape[] | null;
}

export const shapeStandardVersionTranslation: ShapeModel<StandardVersionTranslationShape, StandardVersionTranslationCreateInput, StandardVersionTranslationUpdateInput> = {
    create: (d) => createPrims(d, 'id', 'language', 'description', 'jsonVariable', 'name'),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, 'id', 'description', 'jsonVariable', 'name'), a)
}

export const shapeStandardVersion: ShapeModel<StandardVersionShape, StandardVersionCreateInput, StandardVersionUpdateInput> = {
    create: (d) => ({}) as any,
    update: (o, u, a) => shapeUpdate(u, {}, a) as any
}