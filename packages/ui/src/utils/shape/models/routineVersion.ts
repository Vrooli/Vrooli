import { RoutineVersion, RoutineVersionCreateInput, RoutineVersionTranslation, RoutineVersionTranslationCreateInput, RoutineVersionTranslationUpdateInput, RoutineVersionUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { NodeShape, shapeNode } from "./node";
import { NodeLinkShape, shapeNodeLink } from "./nodeLink";
import { ResourceListShape, shapeResourceList } from "./resourceList";
import { RoutineShape, shapeRoutine } from "./routine";
import { RoutineVersionInputShape, shapeRoutineVersionInput } from "./routineVersionInput";
import { RoutineVersionOutputShape, shapeRoutineVersionOutput } from "./routineVersionOutput";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel, updateTranslationPrims } from "./tools";

export type RoutineVersionTranslationShape = Pick<RoutineVersionTranslation, "id" | "language" | "description" | "instructions" | "name"> & {
    __typename?: "RoutineVersionTranslation";
}

export type RoutineVersionShape = Pick<RoutineVersion, "id" | "isAutomatable" | "isComplete" | "isPrivate" | "versionLabel" | "versionNotes" | "smartContractCallData"> & {
    __typename: "RoutineVersion";
    apiVersion?: { id: string } | null;
    directoryListings?: { id: string }[] | null;
    inputs?: RoutineVersionInputShape[] | null;
    nodes?: NodeShape[] | null;
    nodeLinks?: NodeLinkShape[] | null;
    outputs?: RoutineVersionOutputShape[] | null;
    resourceList?: ResourceListShape | null;
    root?: { __typename: "Routine", id: string } | RoutineShape | null;
    smartContractVersion?: { id: string } | null;
    suggestedNextByRoutineVersion?: { id: string }[] | null;
    translations?: RoutineVersionTranslationShape[] | null;
}

export const shapeRoutineVersionTranslation: ShapeModel<RoutineVersionTranslationShape, RoutineVersionTranslationCreateInput, RoutineVersionTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", "instructions", "name"),
    update: (o, u, a) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description", "instructions", "name"), a),
};

export const shapeRoutineVersion: ShapeModel<RoutineVersionShape, RoutineVersionCreateInput, RoutineVersionUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "isAutomatable", "isComplete", "isPrivate", "versionLabel", "versionNotes", "smartContractCallData");
        return {
            ...prims,
            ...createRel(d, "apiVersion", ["Connect"], "one"),
            ...createRel(d, "directoryListings", ["Connect"], "many"),
            ...createRel(d, "inputs", ["Create"], "many", shapeRoutineVersionInput, (i) => ({ ...i, routineVersion: { id: prims.id } })),
            ...createRel(d, "nodes", ["Create"], "many", shapeNode, (n) => ({ ...n, routineVersion: { id: prims.id } })),
            ...createRel(d, "nodeLinks", ["Create"], "many", shapeNodeLink, (nl) => ({ ...nl, routineVersion: { id: prims.id } })),
            ...createRel(d, "outputs", ["Create"], "many", shapeRoutineVersionOutput, (out) => ({ ...out, routineVersion: { id: prims.id } })),
            ...createRel(d, "resourceList", ["Create"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: prims.id, __typename: "RoutineVersion" } })),
            ...createRel(d, "root", ["Connect", "Create"], "one", shapeRoutine, (r) => ({ ...r, isPrivate: d.isPrivate })),
            ...createRel(d, "smartContractVersion", ["Connect"], "one"),
            ...createRel(d, "suggestedNextByRoutineVersion", ["Connect"], "many"),
            ...createRel(d, "translations", ["Create"], "many", shapeRoutineVersionTranslation),
        };
    },
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isAutomatable", "isComplete", "isPrivate", "versionLabel", "versionNotes", "smartContractCallData"),
        ...updateRel(o, u, "apiVersion", ["Connect", "Disconnect"], "one"),
        ...updateRel(o, u, "directoryListings", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "inputs", ["Create", "Update", "Delete"], "many", shapeRoutineVersionInput, (i) => ({ ...i, routineVersion: { id: o.id } })),
        ...updateRel(o, u, "nodes", ["Create", "Update", "Delete"], "many", shapeNode, (n) => ({ ...n, routineVersion: { id: o.id } })),
        ...updateRel(o, u, "nodeLinks", ["Create", "Update", "Delete"], "many", shapeNodeLink, (nl) => ({ ...nl, routineVersion: { id: o.id } })),
        ...updateRel(o, u, "outputs", ["Create", "Update", "Delete"], "many", shapeRoutineVersionOutput, (out) => ({ ...out, routineVersion: { id: o.id } })),
        ...updateRel(o, u, "resourceList", ["Create", "Update"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: o.id, __typename: "RoutineVersion" } })),
        ...updateRel(o, u, "root", ["Update"], "one", shapeRoutine),
        ...updateRel(o, u, "smartContractVersion", ["Connect", "Disconnect"], "one"),
        ...updateRel(o, u, "suggestedNextByRoutineVersion", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeRoutineVersionTranslation),
    }, a),
};
