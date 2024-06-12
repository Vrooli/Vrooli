import { ProjectVersionDirectory, ProjectVersionDirectoryCreateInput, ProjectVersionDirectoryUpdateInput } from "@local/shared";
import { CanConnect, ShapeModel } from "types";
import { ApiVersionShape } from "./apiVersion";
import { CodeVersionShape } from "./codeVersion";
import { NoteVersionShape } from "./noteVersion";
import { ProjectVersionShape } from "./projectVersion";
import { RoutineVersionShape } from "./routineVersion";
import { StandardVersionShape } from "./standardVersion";
import { TeamShape } from "./team";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";

export type ProjectVersionDirectoryShape = Pick<ProjectVersionDirectory, "id" | "isRoot" | "childOrder"> & {
    __typename: "ProjectVersionDirectory";
    childApiVersions?: CanConnect<ApiVersionShape>[] | null;
    childCodeVersions?: CanConnect<CodeVersionShape>[] | null;
    childNoteVersions?: CanConnect<NoteVersionShape>[] | null;
    childProjectVersions?: CanConnect<ProjectVersionShape>[] | null;
    childRoutineVersions?: CanConnect<RoutineVersionShape>[] | null;
    childStandardVersions?: CanConnect<StandardVersionShape>[] | null;
    childTeams?: CanConnect<TeamShape>[] | null;
    parentDirectory?: CanConnect<ProjectVersionDirectoryShape> | null;
    projectVersion?: CanConnect<ProjectVersionShape> | null;
}

export const shapeProjectVersionDirectory: ShapeModel<ProjectVersionDirectoryShape, ProjectVersionDirectoryCreateInput, ProjectVersionDirectoryUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "isRoot", "childOrder"),
        ...createRel(d, "childApiVersions", ["Connect"], "many"),
        ...createRel(d, "childCodeVersions", ["Connect"], "many"),
        ...createRel(d, "childNoteVersions", ["Connect"], "many"),
        ...createRel(d, "childProjectVersions", ["Connect"], "many"),
        ...createRel(d, "childRoutineVersions", ["Connect"], "many"),
        ...createRel(d, "childStandardVersions", ["Connect"], "many"),
        ...createRel(d, "childTeams", ["Connect"], "many"),
        ...createRel(d, "parentDirectory", ["Connect"], "one"),
        ...createRel(d, "projectVersion", ["Connect"], "one"),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "isRoot", "childOrder"),
        ...updateRel(o, u, "childApiVersions", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "childCodeVersions", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "childNoteVersions", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "childProjectVersions", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "childRoutineVersions", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "childStandardVersions", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "childTeams", ["Connect", "Disconnect"], "many"),
        ...updateRel(o, u, "parentDirectory", ["Connect", "Disconnect"], "one"),
        ...updateRel(o, u, "projectVersion", ["Connect", "Disconnect"], "one"),
    }, a),
};
