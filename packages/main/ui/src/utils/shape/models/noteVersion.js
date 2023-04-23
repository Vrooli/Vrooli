import { shapeNote } from "./note";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";
export const shapeNoteVersionTranslation = {
    create: (d) => createPrims(d, "id", "language", "description", "name", "text"),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, "id", "language", "description", "name", "text"), a),
};
export const shapeNoteVersion = {
    create: (d) => ({
        ...createPrims(d, "id", "isPrivate", "versionLabel", "versionNotes"),
        ...createRel(d, "directoryListings", ["Connect"], "many"),
        ...createRel(d, "root", ["Connect", "Create"], "one", shapeNote),
        ...createRel(d, "translations", ["Create"], "many", shapeNoteVersionTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isPrivate", "versionLabel", "versionNotes"),
        ...updateRel(o, u, "directoryListings", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "root", ["Update"], "one", shapeNote),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeNoteVersionTranslation),
    }, a),
};
//# sourceMappingURL=noteVersion.js.map