import { shapeResource } from "./resource";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";
export const shapeResourceListTranslation = {
    create: (d) => createPrims(d, "id", "language", "description", "name"),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, "id", "description", "name")),
};
export const shapeResourceList = {
    create: (d) => ({
        ...createPrims(d, "id"),
        ...createRel(d, "resources", ["Create"], "many", shapeResource, (r) => ({ list: { id: d.id }, ...r })),
        ...createRel(d, "translations", ["Create"], "many", shapeResourceListTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id"),
        ...updateRel(o, u, "resources", ["Create", "Update", "Delete"], "many", shapeResource, (r, i) => ({ list: { id: i.id }, ...r })),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeResourceListTranslation),
    }),
};
//# sourceMappingURL=resourceList.js.map