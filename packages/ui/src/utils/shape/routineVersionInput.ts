import { RoutineVersionInput, RoutineVersionInputCreateInput, RoutineVersionInputTranslation, RoutineVersionInputTranslationCreateInput, RoutineVersionInputTranslationUpdateInput, RoutineVersionInputUpdateInput } from "@shared/consts";
import { OmitCalculated } from "types";
import { hasObjectChanged, shapeStandardCreate, StandardShape } from "utils";
import { shapeCreateList, shapeCreatePrims, shapeUpdatePrims, shapeUpdate, shapeUpdateList } from "./shapeTools";

export type RoutineVersionInputTranslationShape = OmitCalculated<RoutineVersionInputTranslation>

export type RoutineVersionInputShape = Omit<OmitCalculated<RoutineVersionInput>, 'translations' | 'standard'> & {
    id: string;
    translations: RoutineVersionInputTranslationShape[];
    standard: StandardShape | null;
}

export const shapeInputTranslationCreate = (item: RoutineVersionInputTranslationShape): RoutineVersionInputTranslationCreateInput =>
    shapeCreatePrims(item, 'id', 'language', 'description', 'helpText')

export const shapeInputTranslationUpdate = (o: RoutineVersionInputTranslationShape, u: RoutineVersionInputTranslationShape): RoutineVersionInputTranslationUpdateInput | undefined =>
    shapeUpdate(u, shapeUpdatePrims(o, u, 'id', 'description', 'helpText'))

export const shapeInputCreate = (item: RoutineVersionInputShape): RoutineVersionInputCreateInput => {
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
}

export const shapeInputUpdate = (o: RoutineVersionInputShape, u: RoutineVersionInputShape): RoutineVersionInputUpdateInput | undefined =>
    shapeUpdate(u, () => {
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