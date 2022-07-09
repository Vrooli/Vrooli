import { OutputItemCreateInput, OutputItemTranslationCreateInput, OutputItemTranslationUpdateInput, OutputItemUpdateInput } from "graphql/generated/globalTypes";
import { RoutineInput, RoutineOutputTranslation, ShapeWrapper } from "types";
import { hasObjectChanged, shapeStandardCreate, StandardShape } from "utils";
import { shapeCreateList, shapeUpdate, shapeUpdateList } from "./shapeTools";

export type OutputTranslationShape = Omit<ShapeWrapper<RoutineOutputTranslation>, 'language' | 'description'> & {
    id: string;
    language: OutputItemTranslationCreateInput['language'];
    description: OutputItemTranslationCreateInput['description'];
}

export type OutputShape = Omit<ShapeWrapper<RoutineInput>, 'translations' | 'standard'> & {
    id: string;
    translations: OutputTranslationShape[];
    standard: StandardShape | null;
}

export const shapeOutputTranslationCreate = (item: OutputTranslationShape): OutputItemTranslationCreateInput => ({
    id: item.id,
    language: item.language,
    description: item.description,
})

export const shapeOutputTranslationUpdate = (
    original: OutputTranslationShape,
    updated: OutputTranslationShape
): OutputItemTranslationUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: u.id,
        description: u.description !== o.description ? u.description : undefined,
    }), 'id')

export const shapeOutputCreate = (item: OutputShape): OutputItemCreateInput => {
    // Connect to standard if it's marked as external. 
    // Otherwise, set as create. The backend will handle the rest
    const shouldConnectToStandard = item.standard && !item.standard.isInternal && item.standard.id;
    return {
        id: item.id,
        name: item.name,
        standardConnect: shouldConnectToStandard ? item.standard?.id as string : undefined,
        standardCreate: !shouldConnectToStandard ? shapeStandardCreate(item.standard as StandardShape) : undefined,
        ...shapeCreateList(item, 'translations', shapeOutputTranslationCreate),
    }
}

export const shapeOutputUpdate = (
    original: OutputShape,
    updated: OutputShape
): OutputItemUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => {
        // Connect to standard if it's marked as external. 
        // Otherwise, set as create. The backend will handle the rest, even if 
        // you're updating.
        const shouldConnectToStandard = u.standard && !u.standard.isInternal && u.standard.id;
        return {
            id: o.id,
            name: u.name,
            standardConnect: shouldConnectToStandard ? u.standard?.id as string : undefined,
            standardCreate: !shouldConnectToStandard ? shapeStandardCreate(u.standard as StandardShape) : undefined,
            ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeOutputTranslationCreate, shapeOutputTranslationUpdate, 'id'),
        }
    }, 'id')