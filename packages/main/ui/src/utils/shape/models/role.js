import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";
export const shapeRoleTranslation = {
    create: (d) => createPrims(d, "id", "language", "description"),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, "id", "description")),
};
export const shapeRole = {
    create: (d) => ({
        ...createPrims(d, "id", "name", "permissions"),
        ...createRel(d, "members", ["Connect"], "many"),
        ...createRel(d, "organization", ["Connect"], "one"),
        ...createRel(d, "translations", ["Create"], "many", shapeRoleTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "name", "permissions"),
        ...updateRel(o, u, "members", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeRoleTranslation),
    }),
};
//# sourceMappingURL=role.js.map