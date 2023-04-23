import { shapeLabel } from "./label";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";
export const shapeIssueTranslation = {
    create: (d) => createPrims(d, "id", "language", "description", "name"),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, "id", "description", "name"), a),
};
export const shapeIssue = {
    create: (d) => ({
        ...createPrims(d, "id", "issueFor"),
        ...createRel(d, "for", ["Connect"], "one"),
        ...createRel(d, "labels", ["Connect", "Create"], "many", shapeLabel),
        ...createRel(d, "translations", ["Create"], "many", shapeIssueTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id"),
        ...updateRel(o, u, "labels", ["Connect", "Disconnect", "Create"], "many", shapeLabel),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeIssueTranslation),
    }, a),
};
//# sourceMappingURL=issue.js.map