import { shapeNode } from "./node";
import { shapeNodeLink } from "./nodeLink";
import { shapeResourceList } from "./resourceList";
import { shapeRoutine } from "./routine";
import { shapeRoutineVersionInput } from "./routineVersionInput";
import { shapeRoutineVersionOutput } from "./routineVersionOutput";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";
export const shapeRoutineVersionTranslation = {
    create: (d) => createPrims(d, "id", "language", "description", "instructions", "name"),
    update: (o, u, a) => shapeUpdate(u, updatePrims(o, u, "id", "description", "instructions", "name"), a),
};
export const shapeRoutineVersion = {
    create: (d) => ({
        ...createPrims(d, "id", "isAutomatable", "isComplete", "isPrivate", "versionLabel", "versionNotes", "smartContractCallData"),
        ...createRel(d, "apiVersion", ["Connect"], "one"),
        ...createRel(d, "directoryListings", ["Connect"], "many"),
        ...createRel(d, "inputs", ["Create"], "many", shapeRoutineVersionInput, (i) => ({ ...i, routineVersion: { id: d.id } })),
        ...createRel(d, "nodes", ["Create"], "many", shapeNode, (n) => ({ ...n, routineVersion: { id: d.id } })),
        ...createRel(d, "nodeLinks", ["Create"], "many", shapeNodeLink, (nl) => ({ ...nl, routineVersion: { id: d.id } })),
        ...createRel(d, "outputs", ["Create"], "many", shapeRoutineVersionOutput, (out) => ({ ...out, routineVersion: { id: d.id } })),
        ...createRel(d, "resourceList", ["Create"], "one", shapeResourceList),
        ...createRel(d, "root", ["Connect", "Create"], "one", shapeRoutine),
        ...createRel(d, "smartContractVersion", ["Connect"], "one"),
        ...createRel(d, "suggestedNextByRoutineVersion", ["Connect"], "many"),
        ...createRel(d, "translations", ["Create"], "many", shapeRoutineVersionTranslation),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isAutomatable", "isComplete", "isPrivate", "versionLabel", "versionNotes", "smartContractCallData"),
        ...updateRel(o, u, "apiVersion", ["Connect", "Disconnect"], "one"),
        ...updateRel(o, u, "directoryListings", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "inputs", ["Create", "Update", "Delete"], "many", shapeRoutineVersionInput, (i) => ({ ...i, routineVersion: { id: o.id } })),
        ...updateRel(o, u, "nodes", ["Create", "Update", "Delete"], "many", shapeNode, (n) => ({ ...n, routineVersion: { id: o.id } })),
        ...updateRel(o, u, "nodeLinks", ["Create", "Update", "Delete"], "many", shapeNodeLink, (nl) => ({ ...nl, routineVersion: { id: o.id } })),
        ...updateRel(o, u, "outputs", ["Create", "Update", "Delete"], "many", shapeRoutineVersionOutput, (out) => ({ ...out, routineVersion: { id: o.id } })),
        ...updateRel(o, u, "resourceList", ["Create", "Update"], "one", shapeResourceList),
        ...updateRel(o, u, "root", ["Update"], "one", shapeRoutine),
        ...updateRel(o, u, "smartContractVersion", ["Connect", "Disconnect"], "one"),
        ...updateRel(o, u, "suggestedNextByRoutineVersion", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeRoutineVersionTranslation),
    }, a),
};
//# sourceMappingURL=routineVersion.js.map