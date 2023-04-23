import { shapeTag } from "./tag";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";
export const shapeQuestionTranslation = {
    create: (d) => createPrims(d, "id", "language", "description", "name"),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, "id", "language", "description", "name"), a),
};
export const shapeQuestion = {
    create: (d) => ({
        forObjectConnect: d.forObject?.id ?? undefined,
        forObjectType: d.forObject?.__typename ?? undefined,
        ...createPrims(d, "id", "isPrivate", "referencing"),
        ...createRel(d, "tags", ["Connect", "Create"], "many", shapeTag),
        ...createRel(d, "translations", ["Create"], "many", shapeQuestionTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isPrivate"),
        ...updateRel(o, u, "acceptedAnswer", ["Connect", "Disconnect"], "one"),
        ...updateRel(o, u, "tags", ["Connect", "Create", "Disconnect"], "many", shapeTag),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeQuestionTranslation),
    }, a),
};
//# sourceMappingURL=question.js.map