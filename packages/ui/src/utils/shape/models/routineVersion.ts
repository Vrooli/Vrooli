import { RoutineVersion, RoutineVersionCreateInput, RoutineVersionTranslation, RoutineVersionTranslationCreateInput, RoutineVersionTranslationUpdateInput, RoutineVersionUpdateInput } from "@local/shared";
import { CanConnect, ShapeModel } from "types";
import { ApiVersionShape } from "./apiVersion";
import { CodeVersionShape } from "./codeVersion";
import { NodeShape, shapeNode } from "./node";
import { NodeLinkShape, shapeNodeLink } from "./nodeLink";
import { ProjectVersionDirectoryShape } from "./projectVersionDirectory";
import { ResourceListShape, shapeResourceList } from "./resourceList";
import { RoutineShape, shapeRoutine } from "./routine";
import { RoutineVersionInputShape, shapeRoutineVersionInput } from "./routineVersionInput";
import { RoutineVersionOutputShape, shapeRoutineVersionOutput } from "./routineVersionOutput";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel, updateTranslationPrims } from "./tools";

export type RoutineVersionTranslationShape = Pick<RoutineVersionTranslation, "id" | "language" | "description" | "instructions" | "name"> & {
    __typename?: "RoutineVersionTranslation";
}

export type RoutineVersionShape = Pick<RoutineVersion, "id" | "isAutomatable" | "isComplete" | "isPrivate" | "versionLabel" | "versionNotes" | "codeCallData"> & {
    __typename: "RoutineVersion";
    apiVersion?: CanConnect<ApiVersionShape> | null;
    codeVersion?: CanConnect<CodeVersionShape> | null;
    directoryListings?: CanConnect<ProjectVersionDirectoryShape>[] | null;
    inputs?: RoutineVersionInputShape[] | null;
    nodes?: NodeShape[] | null;
    nodeLinks?: NodeLinkShape[] | null;
    outputs?: RoutineVersionOutputShape[] | null;
    resourceList?: ResourceListShape | null;
    root?: CanConnect<RoutineShape> | null;
    suggestedNextByRoutineVersion?: CanConnect<RoutineVersionShape>[] | null;
    translations?: RoutineVersionTranslationShape[] | null;
}

export const shapeRoutineVersionTranslation: ShapeModel<RoutineVersionTranslationShape, RoutineVersionTranslationCreateInput, RoutineVersionTranslationUpdateInput> = {
    create: (d) => createPrims(d, "id", "language", "description", ["instructions", (instructions) => instructions ?? ""], "name"),
    update: (o, u, a) => shapeUpdate(u, updateTranslationPrims(o, u, "id", "description", "instructions", "name"), a),
};

export const shapeRoutineVersion: ShapeModel<RoutineVersionShape, RoutineVersionCreateInput, RoutineVersionUpdateInput> = {
    create: (d) => {
        const prims = createPrims(d, "id", "codeCallData", "isAutomatable", "isComplete", "isPrivate", "versionLabel", "versionNotes");
        return {
            ...prims,
            ...createRel(d, "apiVersion", ["Connect"], "one"),
            ...createRel(d, "codeVersion", ["Connect"], "one"),
            ...createRel(d, "directoryListings", ["Connect"], "many"),
            ...createRel(d, "inputs", ["Create"], "many", shapeRoutineVersionInput, (i) => ({ ...i, routineVersion: { id: prims.id } })),
            ...createRel(d, "nodes", ["Create"], "many", shapeNode, (n) => ({ ...n, routineVersion: { id: prims.id } })),
            ...createRel(d, "nodeLinks", ["Create"], "many", shapeNodeLink, (nl) => ({ ...nl, routineVersion: { id: prims.id } })),
            ...createRel(d, "outputs", ["Create"], "many", shapeRoutineVersionOutput, (out) => ({ ...out, routineVersion: { id: prims.id } })),
            ...createRel(d, "resourceList", ["Create"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: prims.id, __typename: "RoutineVersion" } })),
            ...createRel(d, "root", ["Connect", "Create"], "one", shapeRoutine, (r) => ({ ...r, isPrivate: d.isPrivate })),
            ...createRel(d, "suggestedNextByRoutineVersion", ["Connect"], "many"),
            ...createRel(d, "translations", ["Create"], "many", shapeRoutineVersionTranslation),
        };
    },
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "codeCallData", "isAutomatable", "isComplete", "isPrivate", "versionLabel", "versionNotes"),
        ...updateRel(o, u, "apiVersion", ["Connect", "Disconnect"], "one"),
        ...updateRel(o, u, "codeVersion", ["Connect", "Disconnect"], "one"),
        ...updateRel(o, u, "directoryListings", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "inputs", ["Create", "Update", "Delete"], "many", shapeRoutineVersionInput, (i) => ({ ...i, routineVersion: { id: o.id } })),
        ...updateRel(o, u, "nodes", ["Create", "Update", "Delete"], "many", shapeNode, (n) => ({ ...n, routineVersion: { id: o.id } })),
        ...updateRel(o, u, "nodeLinks", ["Create", "Update", "Delete"], "many", shapeNodeLink, (nl) => ({ ...nl, routineVersion: { id: o.id } })),
        ...updateRel(o, u, "outputs", ["Create", "Update", "Delete"], "many", shapeRoutineVersionOutput, (out) => ({ ...out, routineVersion: { id: o.id } })),
        ...updateRel(o, u, "resourceList", ["Create", "Update"], "one", shapeResourceList, (l) => ({ ...l, listFor: { id: o.id, __typename: "RoutineVersion" } })),
        ...updateRel(o, u, "root", ["Update"], "one", shapeRoutine),
        ...updateRel(o, u, "suggestedNextByRoutineVersion", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "translations", ["Create", "Update", "Delete"], "many", shapeRoutineVersionTranslation),
    }, a),
};
