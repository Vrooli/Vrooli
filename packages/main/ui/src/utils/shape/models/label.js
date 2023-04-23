import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";
export const shapeLabelTranslation = {
    create: (d) => createPrims(d, "id", "language", "description"),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, "id", "description"), a),
};
export const shapeLabel = {
    create: (d) => ({
        ...createPrims(d, "id", "label", "color"),
        ...createRel(d, "organization", ["Connect"], "one"),
        ...createRel(d, "translations", ["Create"], "many", shapeLabelTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "label", "color"),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeLabelTranslation),
    }, a),
};
//# sourceMappingURL=label.js.map