import { InputItemCreateInput, InputItemTranslationCreateInput, InputItemTranslationUpdateInput, InputItemUpdateInput } from "graphql/generated/globalTypes";
import { RoutineInput, RoutineInputTranslation, ShapeWrapper } from "types";
import { hasObjectChanged, shapeStandardCreate, StandardShape } from "utils";
import { shapeCreateList, shapeUpdate, shapeUpdateList } from "./shapeTools";

export type InputTranslationShape = Omit<ShapeWrapper<RoutineInputTranslation>, 'language' | 'description'> & {
    id: string;
    language: InputItemTranslationCreateInput['language'];
    description: InputItemTranslationCreateInput['description'];
}

export type InputShape = Omit<ShapeWrapper<RoutineInput>, 'translations' | 'standard'> & {
    id: string;
    translations: InputTranslationShape[];
    standard: StandardShape | null;
}

export const shapeInputTranslationCreate = (item: InputTranslationShape): InputItemTranslationCreateInput => ({
    id: item.id,
    language: item.language,
    description: item.description,
})

export const shapeInputTranslationUpdate = (
    original: InputTranslationShape,
    updated: InputTranslationShape
): InputItemTranslationUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: u.id,
        description: u.description !== o.description ? u.description : undefined,
    }))

export const shapeInputTranslationsCreate = (items: InputTranslationShape[] | null | undefined): {
    translationsCreate?: InputItemTranslationCreateInput[],
} => shapeCreateList(items, 'translations', shapeInputTranslationCreate);

export const shapeInputTranslationsUpdate = (
    o: InputTranslationShape[] | null | undefined,
    u: InputTranslationShape[] | null | undefined
): {
    translationsCreate?: InputItemTranslationCreateInput[],
    translationsUpdate?: InputItemTranslationUpdateInput[],
    translationsDelete?: string[],
} => shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeInputTranslationCreate, shapeInputTranslationUpdate)

export const shapeInputCreate = (item: InputShape): InputItemCreateInput => {
    // Connect to standard if it's marked as external. 
    // Otherwise, set as create. The backend will handle the rest
    const shouldConnectToStandard = item.standard && !item.standard.isInternal && item.standard.id;
    return {
        id: item.id,
        isRequired: item.isRequired,
        name: item.name,
        standardConnect: shouldConnectToStandard ? item.standard?.id as string : undefined,
        standardCreate: !shouldConnectToStandard ? shapeStandardCreate(item.standard as StandardShape) : undefined,
        ...shapeInputTranslationsCreate(item.translations),
    }
}

export const shapeInputUpdate = (
    original: InputShape,
    updated: InputShape
): InputItemUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => {
        // Connect to standard if it's marked as external. 
        // Otherwise, set as create. The backend will handle the rest, even if 
        // you're updating.
        const shouldConnectToStandard = u.standard && !u.standard.isInternal && u.standard.id;
        return {
            id: o.id,
            isRequired: u.isRequired,
            name: u.name,
            standardConnect: shouldConnectToStandard ? u.standard?.id as string : undefined,
            standardCreate: !shouldConnectToStandard ? shapeStandardCreate(u.standard as StandardShape) : undefined,
            ...shapeInputTranslationsUpdate(o.translations, u.translations),
        }
    })

export const shapeInputsCreate = (items: InputShape[] | null | undefined): {
    itemsCreate?: InputItemCreateInput[],
} => shapeCreateList(items, 'items', shapeInputCreate);

export const shapeInputsUpdate = (
    o: InputShape[] | null | undefined,
    u: InputShape[] | null | undefined
): {
    itemsCreate?: InputItemCreateInput[],
    itemsUpdate?: InputItemUpdateInput[],
    itemsDelete?: string[],
} => shapeUpdateList(o, u, 'items', hasObjectChanged, shapeInputCreate, shapeInputUpdate)