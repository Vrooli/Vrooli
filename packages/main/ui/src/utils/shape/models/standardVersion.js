import { createPrims, shapeUpdate, updatePrims } from "./tools";
export const shapeStandardVersionTranslation = {
    create: (d) => createPrims(d, "id", "language", "description", "jsonVariable", "name"),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, "id", "description", "jsonVariable", "name"), a),
};
export const shapeStandardVersion = {
    create: (d) => ({}),
    update: (o, u, a) => shapeUpdate(u, {}, a),
};
//# sourceMappingURL=standardVersion.js.map