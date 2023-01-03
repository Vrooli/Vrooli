import { RoutineVersionOutput, RoutineVersionOutputCreateInput, RoutineVersionOutputTranslation, RoutineVersionOutputTranslationCreateInput, RoutineVersionOutputTranslationUpdateInput, RoutineVersionOutputUpdateInput } from "@shared/consts";
import { createPrims, hasObjectChanged, shapeStandardCreate, shapeUpdate, StandardShape, updatePrims } from "utils";

export type RoutineVersionOutputTranslationShape = Pick<RoutineVersionOutputTranslation, 'id' | 'language' | 'description' | 'helpText'>

export type OutputShape = Omit<OmitCalculated<RoutineVersionOutput>, 'translations' | 'standard'> & {
    id: string;
    translations: RoutineVersionOutputTranslationShape[];
    standard: StandardShape | null;
}

export const shapeRoutineVersionOutputTranslation: ShapeModel<RoutineVersionOutputTranslationShape, RoutineVersionOutputTranslationCreateOutput, RoutineVersionInputTranslationUpdateInput> = {
    create: (item) => createPrims(item, 'id', 'language', 'description', 'helpText'),
    update: (o, u) => shapeUpdate(u, updatePrims(o, u, 'id', 'description', 'helpText'))
}

export const shapeRoutineVersionOutput: ShapeModel<RoutineVersionOutputShape, RoutineVersionOutputCreateInput, RoutineVersionOutputUpdateInput> = {
    create: (item) => {
        // Connect to standard if it's marked as external. 
        // Otherwise, set as create. The backend will handle the rest
        const shouldConnectToStandard = item.standard && !item.standard.isInternal && item.standard.id;
        return {
            id: item.id,
            name: item.name,
            standardVersionConnect: {} as any,//TODO shouldConnectToStandard ? item.standard?.id as string : undefined,
            standardCreate: item.standard && !shouldConnectToStandard ? shapeStandardCreate(item.standard) : undefined,
            ...shapeCreateList(item, 'translations', shapeOutputTranslationCreate),
        }
    },
    update: (o, u) => shapeUpdate(u, () => {
        // Connect to standard if it's marked as external. 
        // Otherwise, set as create. The backend will handle the rest, even if 
        // you're updating.
        const shouldConnectToStandard = u.standard && !u.standard.isInternal && u.standard.id;
        return {
            id: o.id,
            name: u.name,
            standardConnect: shouldConnectToStandard ? u.standard?.id as string : undefined,
            standardCreate: u.standard && !shouldConnectToStandard ? shapeStandardCreate(u.standard as StandardShape) : undefined,
            ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeOutputTranslationCreate, shapeOutputTranslationUpdate, 'id'),
        }
    })
}
    