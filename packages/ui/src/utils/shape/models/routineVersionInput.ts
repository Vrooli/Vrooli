import { RoutineVersionInput, RoutineVersionInputCreateInput, RoutineVersionInputTranslation, RoutineVersionInputTranslationCreateInput, RoutineVersionInputTranslationUpdateInput, RoutineVersionInputUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { createPrims, createRel, hasObjectChanged, shapeStandardVersion, shapeUpdate, StandardVersionShape, updatePrims, updateRel } from "utils";

export type RoutineVersionInputTranslationShape = Pick<RoutineVersionInputTranslation, 'id' | 'language' | 'description' | 'helpText'>

export type RoutineVersionInputShape = Pick<RoutineVersionInput, 'id' | 'index' | 'isRequired' | 'name'> & {
    routineVersion: { id: string };
    standardVersion?: StandardVersionShape | null;
    translations?: RoutineVersionInputTranslationShape[];
}

export const shapeRoutineVersionInputTranslation: ShapeModel<RoutineVersionInputTranslationShape, RoutineVersionInputTranslationCreateInput, RoutineVersionInputTranslationUpdateInput> = {
    create: (d) => createPrims(d, 'id', 'language', 'description', 'helpText'),
    update: (o, u) => shapeUpdate(u, updatePrims(o, u, 'id', 'description', 'helpText'))
}

export const shapeRoutineVersionInput: ShapeModel<RoutineVersionInputShape, RoutineVersionInputCreateInput, RoutineVersionInputUpdateInput> = {
    create: (d) => {
        // Connect to standard if it's marked as external. 
        // Otherwise, set as create. The backend will handle the rest
        const shouldConnectToStandard = d.standardVersion && !d.standardVersion.root.isInternal && d.standardVersion.id;
        return {
            ...createPrims(d, 'id', 'index', 'isRequired', 'name'),
            ...createRel(d, 'routineVersion', ['Connect'], 'one'),
            standardVersionConnect: shouldConnectToStandard ? d.standardVersion!.id : undefined,
            standardCreate: d.standardVersion && !shouldConnectToStandard ? shapeStandardVersion.create(d.standardVersion) : undefined,
            ...createRel(d, 'translations', ['Create'], 'many', shapeRoutineVersionInputTranslation),
        }
    },
    update: (o, u) => shapeUpdate(u, () => {
        // Connect to standard if it's marked as external. 
        // Otherwise, set as create. The backend will handle the rest
        const shouldConnectToStandard = u.standardVersion && !u.standardVersion.root.isInternal && u.standardVersion.id;
        const hasStandardChanged = hasObjectChanged(o.standardVersion, u.standardVersion);
        return {
            ...updatePrims(o, u, 'id', 'index', 'isRequired', 'name'),
            standardConnect: (hasStandardChanged && shouldConnectToStandard) ? u.standardVersion!.id : undefined,
            standardCreate: (u.standardVersion && hasStandardChanged && !shouldConnectToStandard) ? shapeStandardVersion.update(o.standardVersion!, u.standardVersion!) : undefined,
            ...updateRel(o, u, 'translations', ['Create', 'Update', 'Delete'], 'many', shapeRoutineVersionInputTranslation),
        }
    })
}