import { addHttps } from "@local/validation";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";
export const shapeResourceTranslation = {
    create: (d) => createPrims(d, "id", "language", "description", "name"),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, "id", "description", "name")),
};
export const shapeResource = {
    create: (d) => ({
        link: addHttps(d.link),
        ...createPrims(d, "id", "index", "usedFor"),
        ...createRel(d, "list", ["Connect"], "one"),
        ...createRel(d, "translations", ["Create"], "many", shapeResourceTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        link: o.link !== u.link ? addHttps(u.link) : undefined,
        ...updatePrims(o, u, "id", "index", "usedFor"),
        ...updateRel(o, u, "list", ["Connect"], "one"),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeResourceTranslation),
    }),
};
//# sourceMappingURL=resource.js.map