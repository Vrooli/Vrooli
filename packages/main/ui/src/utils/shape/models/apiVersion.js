import { shapeApi } from "./api";
import { shapeResourceList } from "./resourceList";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";
export const shapeApiVersionTranslation = {
    create: (d) => createPrims(d, "id", "language", "details", "name", "summary"),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, "id", "details", "summary"), a),
};
export const shapeApiVersion = {
    create: (d) => ({
        ...createPrims(d, "id", "callLink", "documentationLink", "isComplete", "isPrivate", "versionLabel", "versionNotes"),
        ...createRel(d, "directoryListings", ["Connect"], "many"),
        ...createRel(d, "resourceList", ["Create"], "one", shapeResourceList),
        ...createRel(d, "root", ["Connect", "Create"], "one", shapeApi),
        ...createRel(d, "translations", ["Create"], "many", shapeApiVersionTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "callLink", "documentationLink", "isComplete", "isPrivate", "versionLabel", "versionNotes"),
        ...updateRel(o, u, "directoryListings", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "resourceList", ["Create", "Update"], "one", shapeResourceList),
        ...updateRel(o, u, "root", ["Update"], "one", shapeApi),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeApiVersionTranslation),
    }, a),
};
//# sourceMappingURL=apiVersion.js.map