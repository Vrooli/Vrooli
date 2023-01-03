import { RoutineVersionInput, RoutineVersionInputCreateInput, RoutineVersionInputTranslation, RoutineVersionInputTranslationCreateInput, RoutineVersionInputTranslationUpdateInput, RoutineVersionInputUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { createPrims, hasObjectChanged, shapeStandardCreate, shapeUpdate, StandardShape, updatePrims } from "utils";

export type RoutineVersionInputTranslationShape = Pick<RoutineVersionInputTranslation, 'id' | 'language' | 'description' | 'helpText'>

export type RoutineVersionInputShape = Omit<OmitCalculated<RoutineVersionInput>, 'translations' | 'standard'> & {
    id: string;
    translations: RoutineVersionInputTranslationShape[];
    standard: StandardShape | null;
}

export const shapeRoutineVersionInputTranslation: ShapeModel<RoutineVersionInputTranslationShape, RoutineVersionInputTranslationCreateInput, RoutineVersionInputTranslationUpdateInput> = {
    create: (item) => createPrims(item, 'id', 'language', 'description', 'helpText'),
    update: (o, u) => shapeUpdate(u, updatePrims(o, u, 'id', 'description', 'helpText'))
}

export const shapeRoutineVersionInput: ShapeModel<RoutineVersionInputShape, RoutineVersionInputCreateInput, RoutineVersionInputUpdateInput> = {
    create: (item) => {
        // Connect to standard if it's marked as external. 
        // Otherwise, set as create. The backend will handle the rest
        const shouldConnectToStandard = item.standard && !item.standard.isInternal && item.standard.id;
        return {
            id: item.id,
            isRequired: item.isRequired,
            name: item.name,
            standardVersionConnect: {} as any,//TODO shouldConnectToStandard ? item.standard?.id as string : undefined,
            standardCreate: item.standard && !shouldConnectToStandard ? shapeStandardCreate(item.standard) : undefined,
            ...shapeCreateList(item, 'translations', shapeInputTranslationCreate),
        }
    },
    update: (o, u) => shapeUpdate(u, () => {
        // Connect to standard if it's marked as external. 
        // Otherwise, set as create. The backend will handle the rest, even if 
        // you're updating.
        const shouldConnectToStandard = u.standard && !u.standard.isInternal && u.standard.id;
        return {
            id: o.id,
            isRequired: u.isRequired,
            name: u.name,
            standardConnect: shouldConnectToStandard ? u.standard?.id as string : undefined,
            standardCreate: u.standard && !shouldConnectToStandard ? shapeStandardCreate(u.standard as StandardShape) : undefined,
            ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeInputTranslationCreate, shapeInputTranslationUpdate, 'id'),
        }
    })
}