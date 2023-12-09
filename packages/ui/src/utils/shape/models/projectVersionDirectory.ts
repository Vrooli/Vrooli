import { ProjectVersionDirectory, ProjectVersionDirectoryCreateInput, ProjectVersionDirectoryUpdateInput } from "@local/shared";
import { CanConnect, ShapeModel } from "types";
import { ApiVersionShape } from "./apiVersion";
import { NoteVersionShape } from "./noteVersion";
import { OrganizationShape } from "./organization";
import { ProjectVersionShape } from "./projectVersion";
import { RoutineVersionShape } from "./routineVersion";
import { SmartContractVersionShape } from "./smartContractVersion";
import { StandardVersionShape } from "./standardVersion";
import { createPrims, createRel, shapeUpdate, updatePrims, updateRel } from "./tools";

export type ProjectVersionDirectoryShape = Pick<ProjectVersionDirectory, "id" | "isRoot" | "childOrder"> & {
    __typename: "ProjectVersionDirectory";
    childApiVersions?: CanConnect<ApiVersionShape>[] | null;
    childNoteVersions?: CanConnect<NoteVersionShape>[] | null;
    childOrganizations?: CanConnect<OrganizationShape>[] | null;
    childProjectVersions?: CanConnect<ProjectVersionShape>[] | null;
    childRoutineVersions?: CanConnect<RoutineVersionShape>[] | null;
    childSmartContractVersions?: CanConnect<SmartContractVersionShape>[] | null;
    childStandardVersions?: CanConnect<StandardVersionShape>[] | null;
    parentDirectory?: CanConnect<ProjectVersionDirectoryShape> | null;
    projectVersion?: CanConnect<ProjectVersionShape> | null;
}

export const shapeProjectVersionDirectory: ShapeModel<ProjectVersionDirectoryShape, ProjectVersionDirectoryCreateInput, ProjectVersionDirectoryUpdateInput> = {
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
