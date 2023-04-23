import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";
export const shapeNodeLinkWhenTranslation = {
    create: (d) => createPrims(d, "id", "language", "description", "name"),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, "id", "description", "name"), a),
};
export const shapeNodeLinkWhen = {
    create: (d) => ({
        ...createPrims(d, "id", "condition"),
        ...createRel(d, "link", ["Connect"], "one"),
        ...createRel(d, "translations", ["Create"], "many", shapeNodeLinkWhenTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "condition"),
        ...updateRel(o, u, "link", ["Connect"], "one"),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeNodeLinkWhenTranslation),
    }, a),
};
//# sourceMappingURL=nodeLinkWhen.js.map