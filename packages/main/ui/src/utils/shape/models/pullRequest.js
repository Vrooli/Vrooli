import { createPrims, shapeUpdate, updatePrims } from "./tools";
export const shapePullRequestTranslation = {
    create: (d) => createPrims(d, "id", "language", "text"),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, "id", "text"), a),
};
export const shapePullRequest = {
    create: (d) => ({}),
    update: (o, u, a) => shapeUpdate(u, {}, a),
};
//# sourceMappingURL=pullRequest.js.map