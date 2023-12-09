import { RoutineVersionInput, RoutineVersionInputCreateInput, RoutineVersionInputTranslation, RoutineVersionInputTranslationCreateInput, RoutineVersionInputTranslationUpdateInput, RoutineVersionInputUpdateInput } from "@local/shared";
import { CanConnect, ShapeModel } from "types";
import { hasObjectChanged } from "../general";
import { RoutineVersionShape } from "./routineVersion";
import { StandardVersionShape, shapeStandardVersion } from "./standardVersion";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel, updateTranslationPrims } from "./tools";

export type RoutineVersionInputTranslationShape = Pick<RoutineVersionInputTranslation, "id" | "language" | "description" | "helpText"> & {
    __typename?: "RoutineVersionInputTranslation";
}

export type RoutineVersionInputShape = Pick<RoutineVersionInput, "id" | "index" | "isRequired" | "name"> & {
    __typename: "RoutineVersionInput";
    routineVersion: CanConnect<RoutineVersionShape>;
    standardVersion?: StandardVersionShape | null;
    translations?: RoutineVersionInputTranslationShape[] | null;
}

export const shapeRoutineVersionInputTranslation: ShapeModel<RoutineVersionInputTranslationShape, RoutineVersionInputTranslationCreateInput, RoutineVersionInputTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", "helpText"),
    update: (o, u, a) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description", "helpText"), a),
};

export const shapeRoutineVersionInput: ShapeModel<RoutineVersionInputShape, RoutineVersionInputCreateInput, RoutineVersionInputUpdateInput> = {
    create: (d) => {
        // Connect to standard if it's marked as external. 
        // Otherwise, set as create. The backend will handle the rest
        const shouldConnectToStandard = d.standardVersion && !d.standardVersion.root.isInternal && d.standardVersion.id;
        return {
            ...createPrims(d, "id", "index", "isRequired", "name"),
            ...createRel(d, "routineVersion", ["Connect"], "one"),
            standardVersionConnect: shouldConnectToStandard ? d.standardVersion!.id : undefined,
            standardVersionCreate: d.standardVersion && !shouldConnectToStandard ? shapeStandardVersion.create(d.standardVersion) : undefined,
            ...createRel(d, "translations", ["Create"], "many", shapeRoutineVersionInputTranslation),
        };
    },
    update: (o, u, a) => shapeUpdate(u, () => {
        // Connect to standard if it's marked as external. 
        // Otherwise, set as create. The backend will handle the rest
        const shouldConnectToStandard = u.standardVersion && !u.standardVersion.root.isInternal && u.standardVersion.id;
        const hasStandardChanged = hasObjectChanged(o.standardVersion, u.standardVersion);
        return {
            ...updatePrims(o, u, "id", "index", "isRequired", "name"),
            standardVersionConnect: (hasStandardChanged && shouldConnectToStandard) ? u.standardVersion!.id : undefined,
            standardVersionCreate: (u.standardVersion && hasStandardChanged && !shouldConnectToStandard) ? shapeStandardVersion.create(u.standardVersion!) : undefined,
            ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeRoutineVersionInputTranslation),
        };
    }, a),
};
