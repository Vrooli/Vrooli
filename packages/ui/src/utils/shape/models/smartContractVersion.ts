import { SmartContractVersion, SmartContractVersionCreateInput, SmartContractVersionTranslation, SmartContractVersionTranslationCreateInput, SmartContractVersionTranslationUpdateInput, SmartContractVersionUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { ProjectVersionDirectoryShape, shapeProjectVersionDirectory } from "./projectVersionDirectory";
import { ResourceListShape, shapeResourceList } from "./resourceList";
import { shapeSmartContract, SmartContractShape } from "./smartContract";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel, updateTranslationPrims } from "./tools";

export type SmartContractVersionTranslationShape = Pick<SmartContractVersionTranslation, "id" | "language" | "description" | "name" | "jsonVariable"> & {
    __typename?: "SmartContractVersionTranslation";
}

export type SmartContractVersionShape = Pick<SmartContractVersion, "id" | "content" | "contractType" | "default" | "isComplete" | "isPrivate" | "versionLabel" | "versionNotes"> & {
    __typename: "SmartContractVersion";
    directoryListings?: ProjectVersionDirectoryShape[] | null;
    resourceList?: Omit<ResourceListShape, "listFor"> | null;
    root?: { id: string } | SmartContractShape | null;
    suggestedNextBySmartContract?: { id: string }[] | null;
    translations?: SmartContractVersionTranslationShape[] | null;
}

export const shapeSmartContractVersionTranslation: ShapeModel<SmartContractVersionTranslationShape, SmartContractVersionTranslationCreateInput, SmartContractVersionTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", "name", "jsonVariable"),
    update: (o, u, a) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "language", "description", "name", "jsonVariable"), a),
};

export const shapeSmartContractVersion: ShapeModel<SmartContractVersionShape, SmartContractVersionCreateInput, SmartContractVersionUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "content", "contractType", "default", "isComplete", "isPrivate", "versionLabel", "versionNotes");
        return {
            ...prims,
            ...createRel(d, "directoryListings", ["Create"], "many", shapeProjectVersionDirectory),
            ...createRel(d, "root", ["Connect", "Create"], "one", shapeSmartContract, (r) => ({ ...r, isPrivate: prims.isPrivate })),
            ...createRel(d, "resourceList", ["Create"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: prims.id, __typename: "SmartContractVersion" } })),
            ...createRel(d, "translations", ["Create"], "many", shapeSmartContractVersionTranslation),
        };
    },
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "content", "contractType", "default", "isPrivate", "versionLabel", "versionNotes"),
        ...updateRel(o, u, "directoryListings", ["Create", "Update", "Delete"], "many", shapeProjectVersionDirectory),
        ...updateRel(o, u, "root", ["Update"], "one", shapeSmartContract),
        ...updateRel(o, u, "resourceList", ["Create", "Update"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: o.id, __typename: "SmartContractVersion" } })),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeSmartContractVersionTranslation),
    }, a),
};
