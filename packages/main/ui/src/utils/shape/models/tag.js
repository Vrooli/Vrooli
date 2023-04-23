import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";
export const shapeTagTranslation = {
    create: (d) => createPrims(d, "id", "language", "description"),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, "id", "description"), a),
};
export const shapeTag = {
    idField: "tag",
    create: (d) => ({
        tag: d.tag,
        ...createRel(d, "translations", ["Create"], "many", shapeTagTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        tag: o.tag,
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeTagTranslation),
    }, a),
};
//# sourceMappingURL=tag.js.map