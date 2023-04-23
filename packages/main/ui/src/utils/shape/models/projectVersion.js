import { shapeProject } from "./project";
import { shapeProjectVersionDirectory } from "./projectVersionDirectory";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";
export const shapeProjectVersionTranslation = {
    create: (d) => createPrims(d, "id", "language", "description", "name"),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, "id", "description", "name"), a),
};
export const shapeProjectVersion = {
    create: (d) => ({
        ...createPrims(d, "id", "isComplete", "isPrivate", "versionLabel", "versionNotes"),
        ...createRel(d, "directories", ["Create"], "many", shapeProjectVersionDirectory),
        ...createRel(d, "root", ["Connect", "Create"], "one", shapeProject),
        ...createRel(d, "suggestedNextByProject", ["Connect"], "many"),
        ...createRel(d, "translations", ["Create"], "many", shapeProjectVersionTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isPrivate", "versionLabel", "versionNotes"),
        ...updateRel(o, u, "directories", ["Create", "Update", "Delete"], "many", shapeProjectVersionDirectory),
        ...updateRel(o, u, "root", ["Update"], "one", shapeProject),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeProjectVersionTranslation),
        ...updateRel(o, u, "suggestedNextByProject", ["Connect", "Disconnect"], "many"),
    }, a),
};
//# sourceMappingURL=projectVersion.js.map