import { createPrims, shapeUpdate, updatePrims } from "./tools";
export const shapeReport = {
    create: (d) => ({
        createdForConnect: d.createdFor.id,
        createdFor: d.createdFor.__typename,
        reason: d.otherReason ?? d.reason,
        ...createPrims(d, "id", "details", "language"),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        reason: (u.otherReason ?? u.reason) !== o.reason ? (u.otherReason ?? u.reason) : undefined,
        ...updatePrims(o, u, "id", "details", "language"),
    }, a),
};
//# sourceMappingURL=report.js.map