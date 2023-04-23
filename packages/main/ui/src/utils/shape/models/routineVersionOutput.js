import { hasObjectChanged } from "../general";
import { shapeStandardVersion } from "./standardVersion";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";
export const shapeRoutineVersionOutputTranslation = {
    create: (d) => createPrims(d, "id", "language", "description", "helpText"),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, "id", "description", "helpText"), a),
};
export const shapeRoutineVersionOutput = {
    create: (d) => {
        const shouldConnectToStandard = d.standardVersion && !d.standardVersion.root.isInternal && d.standardVersion.id;
        return {
            ...createPrims(d, "id", "index", "name"),
            ...createRel(d, "routineVersion", ["Connect"], "one"),
            standardVersionConnect: shouldConnectToStandard ? d.standardVersion.id : undefined,
            standardVersionCreate: d.standardVersion && !shouldConnectToStandard ? shapeStandardVersion.create(d.standardVersion) : undefined,
            ...createRel(d, "translations", ["Create"], "many", shapeRoutineVersionOutputTranslation),
        };
    },
    update: (o, u, a) => shapeUpdate(u, () => {
        const shouldConnectToStandard = u.standardVersion && !u.standardVersion.root.isInternal && u.standardVersion.id;
        const hasStandardChanged = hasObjectChanged(o.standardVersion, u.standardVersion);
        return {
            ...updatePrims(o, u, "id", "index", "name"),
            standardVersionConnect: (hasStandardChanged && shouldConnectToStandard) ? u.standardVersion.id : undefined,
            standardVersionCreate: (u.standardVersion && hasStandardChanged && !shouldConnectToStandard) ? shapeStandardVersion.create(u.standardVersion) : undefined,
            ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeRoutineVersionOutputTranslation),
        };
    }, a),
};
//# sourceMappingURL=routineVersionOutput.js.map