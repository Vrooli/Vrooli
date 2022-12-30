import { RoutineVersionOutput, RoutineVersionOutputCreateInput, RoutineVersionOutputTranslation, RoutineVersionOutputTranslationCreateInput, RoutineVersionOutputTranslationUpdateInput, RoutineVersionOutputUpdateInput } from "@shared/consts";
import { OmitCalculated } from "types";
import { hasObjectChanged, shapeStandardCreate, StandardShape } from "utils";
import { shapeCreateList, shapeCreatePrims, shapeUpdatePrims, shapeUpdate, shapeUpdateList } from "./shapeTools";

export type RoutineVersionOutputTranslationShape = OmitCalculated<RoutineVersionOutputTranslation>

export type OutputShape = Omit<OmitCalculated<RoutineVersionOutput>, 'translations' | 'standard'> & {
    id: string;
    translations: RoutineVersionOutputTranslationShape[];
    standard: StandardShape | null;
}

export const shapeOutputTranslationCreate = (item: RoutineVersionOutputTranslationShape): RoutineVersionOutputTranslationCreateInput =>
    shapeCreatePrims(item, 'id', 'language', 'description', 'helpText')

export const shapeOutputTranslationUpdate = (o: RoutineVersionOutputTranslationShape, u: RoutineVersionOutputTranslationShape): RoutineVersionOutputTranslationUpdateInput | undefined =>
    shapeUpdate(u, shapeUpdatePrims(o, u, 'id', 'description', 'helpText'))

export const shapeOutputCreate = (item: OutputShape): RoutineVersionOutputCreateInput => {
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
}

export const shapeOutputUpdate = (o: OutputShape, u: OutputShape): RoutineVersionOutputUpdateInput | undefined =>
    shapeUpdate(u, () => {
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