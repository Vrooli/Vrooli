import { shapeProjectVersionDirectory } from "./projectVersionDirectory";
import { shapeResourceList } from "./resourceList";
import { shapeSmartContract } from "./smartContract";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";
export const shapeSmartContractVersionTranslation = {
    create: (d) => createPrims(d, "id", "language", "description", "name", "jsonVariable"),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, "id", "language", "description", "name", "jsonVariable"), a),
};
export const shapeSmartContractVersion = {
    create: (d) => ({
        ...createPrims(d, "id", "content", "contractType", "default", "isComplete", "isPrivate", "versionLabel", "versionNotes"),
        ...createRel(d, "directoryListings", ["Create"], "many", shapeProjectVersionDirectory),
        ...createRel(d, "root", ["Connect", "Create"], "one", shapeSmartContract),
        ...createRel(d, "resourceList", ["Create"], "one", shapeResourceList),
        ...createRel(d, "translations", ["Create"], "many", shapeSmartContractVersionTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "content", "contractType", "default", "isPrivate", "versionLabel", "versionNotes"),
        ...updateRel(o, u, "directoryListings", ["Create", "Update", "Delete"], "many", shapeProjectVersionDirectory),
        ...updateRel(o, u, "root", ["Update"], "one", shapeSmartContract),
        ...updateRel(o, u, "resourceList", ["Create", "Update"], "one", shapeResourceList),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeSmartContractVersionTranslation),
    }, a),
};
//# sourceMappingURL=smartContractVersion.js.map