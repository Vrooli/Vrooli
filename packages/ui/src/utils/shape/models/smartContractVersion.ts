import { SmartContractVersion, SmartContractVersionCreateInput, SmartContractVersionTranslation, SmartContractVersionTranslationCreateInput, SmartContractVersionTranslationUpdateInput, SmartContractVersionUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { ProjectVersionDirectoryShape, shapeProjectVersionDirectory } from "./projectVersionDirectory";
import { ResourceListShape, shapeResourceList } from "./resourceList";
import { shapeSmartContract, SmartContractShape } from "./smartContract";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";
import { updateTranslationPrims } from "./tools/updateTranslationPrims";

export type SmartContractVersionTranslationShape = Pick<SmartContractVersionTranslation, "id" | "language" | "description" | "name" | "jsonVariable"> & {
    __typename?: "SmartContractVersionTranslation";
}

export type SmartContractVersionShape = Pick<SmartContractVersion, "id" | "content" | "contractType" | "default" | "isComplete" | "isPrivate" | "versionLabel" | "versionNotes"> & {
    __typename?: "SmartContractVersion";
    directoryListings?: ProjectVersionDirectoryShape[] | null;
    resourceList?: { id: string } | ResourceListShape | null;
    root?: { id: string } | SmartContractShape | null;
    suggestedNextBySmartContract?: { id: string }[] | null;
    translations?: SmartContractVersionTranslationShape[] | null;
}

export const shapeSmartContractVersionTranslation: ShapeModel<SmartContractVersionTranslationShape, SmartContractVersionTranslationCreateInput, SmartContractVersionTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", "name", "jsonVariable"),
    update: (o, u, a) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "language", "description", "name", "jsonVariable"), a),
};

export const shapeSmartContractVersion: ShapeModel<SmartContractVersionShape, SmartContractVersionCreateInput, SmartContractVersionUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "content", "contractType", "default", "isComplete", "isPrivate", "versionLabel", "versionNotes"),
        ...createRel(d, "directoryListings", ["Create"], "many", shapeProjectVersionDirectory),
        ...createRel(d, "root", ["Connect", "Create"], "one", shapeSmartContract, (r) => ({ ...r, isPrivate: d.isPrivate })),
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
