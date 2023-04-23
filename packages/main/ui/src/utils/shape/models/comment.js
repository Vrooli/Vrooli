import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";
export const shapeCommentTranslation = {
    create: (d) => createPrims(d, "id", "language", "text"),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, "id", "text"), a),
};
export const shapeComment = {
    create: (d) => ({
        ...createPrims(d, "id", "threadId"),
        createdFor: d.commentedOn.__typename,
        forConnect: d.commentedOn.id,
        ...createRel(d, "translations", ["Create"], "many", shapeCommentTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id"),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeCommentTranslation),
    }, a),
};
//# sourceMappingURL=comment.js.map