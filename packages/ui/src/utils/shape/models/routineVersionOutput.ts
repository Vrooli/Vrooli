import { RoutineVersionOutput, RoutineVersionOutputCreateInput, RoutineVersionOutputTranslation, RoutineVersionOutputTranslationCreateInput, RoutineVersionOutputTranslationUpdateInput, RoutineVersionOutputUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { hasObjectChanged } from "../general";
import { shapeStandardVersion, StandardVersionShape } from "./standardVersion";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel, updateTranslationPrims } from "./tools";

export type RoutineVersionOutputTranslationShape = Pick<RoutineVersionOutputTranslation, "id" | "language" | "description" | "helpText"> & {
    __typename?: "RoutineVersionOutputTranslation";
}

export type RoutineVersionOutputShape = Pick<RoutineVersionOutput, "id" | "index" | "name"> & {
    __typename?: "RoutineVersionOutput";
    routineVersion: { id: string };
    standardVersion?: StandardVersionShape | null;
    translations?: RoutineVersionOutputTranslationShape[] | null;
}

export const shapeRoutineVersionOutputTranslation: ShapeModel<RoutineVersionOutputTranslationShape, RoutineVersionOutputTranslationCreateInput, RoutineVersionOutputTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", "helpText"),
    update: (o, u, a) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description", "helpText"), a),
};

export const shapeRoutineVersionOutput: ShapeModel<RoutineVersionOutputShape, RoutineVersionOutputCreateInput, RoutineVersionOutputUpdateInput> = {
    create: (d) => {
        // Connect to standard if it's marked as external. 
        // Otherwise, set as create. The backend will handle the rest
        const shouldConnectToStandard = d.standardVersion && !d.standardVersion.root.isInternal && d.standardVersion.id;
        return {
            ...createPrims(d, "id", "index", "name"),
            ...createRel(d, "routineVersion", ["Connect"], "one"),
            standardVersionConnect: shouldConnectToStandard ? d.standardVersion!.id : undefined,
            standardVersionCreate: d.standardVersion && !shouldConnectToStandard ? shapeStandardVersion.create(d.standardVersion) : undefined,
            ...createRel(d, "translations", ["Create"], "many", shapeRoutineVersionOutputTranslation),
        };
    },
    update: (o, u, a) => shapeUpdate(u, () => {
        // Connect to standard if it's marked as external. 
        // Otherwise, set as create. The backend will handle the rest
        const shouldConnectToStandard = u.standardVersion && !u.standardVersion.root.isInternal && u.standardVersion.id;
        const hasStandardChanged = hasObjectChanged(o.standardVersion, u.standardVersion);
        return {
            ...updatePrims(o, u, "id", "index", "name"),
            standardVersionConnect: (hasStandardChanged && shouldConnectToStandard) ? u.standardVersion!.id : undefined,
            standardVersionCreate: (u.standardVersion && hasStandardChanged && !shouldConnectToStandard) ? shapeStandardVersion.create(u.standardVersion!) : undefined,
            ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeRoutineVersionOutputTranslation),
        };
    }, a),
};
