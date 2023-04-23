import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";
export const shapeProjectVersionDirectory = {
    create: (d) => ({
        ...createPrims(d, "id", "isRoot", "childOrder"),
        ...createRel(d, "childApiVersions", ["Connect"], "many"),
        ...createRel(d, "childNoteVersions", ["Connect"], "many"),
        ...createRel(d, "childOrganizations", ["Connect"], "many"),
        ...createRel(d, "childProjectVersions", ["Connect"], "many"),
        ...createRel(d, "childRoutineVersions", ["Connect"], "many"),
        ...createRel(d, "childSmartContractVersions", ["Connect"], "many"),
        ...createRel(d, "childStandardVersions", ["Connect"], "many"),
        ...createRel(d, "parentDirectory", ["Connect"], "one"),
        ...createRel(d, "projectVersion", ["Connect"], "one"),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isRoot", "childOrder"),
        ...updateRel(o, u, "childApiVersions", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "childNoteVersions", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "childOrganizations", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "childProjectVersions", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "childRoutineVersions", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "childSmartContractVersions", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "childStandardVersions", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "parentDirectory", ["Connect", "Disconnect"], "one"),
        ...updateRel(o, u, "projectVersion", ["Connect", "Disconnect"], "one"),
    }, a),
};
//# sourceMappingURL=projectVersionDirectory.js.map