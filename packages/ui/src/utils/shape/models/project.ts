import { Project, ProjectCreateInput, ProjectUpdateInput } from "@local/shared";
import { ShapeModel } from "types";
import { LabelShape, shapeLabel } from "./label";
import { ProjectVersionShape, shapeProjectVersion } from "./projectVersion";
import { shapeTag, TagShape } from "./tag";
import { createOwner, createPrims, createRel, createVersion, shapeUpdate, updateOwner, updatePrims, updateRel, updateVersion } from "./tools";
import { OwnerShape } from "./types";

export type ProjectShape = Pick<Project, "id" | "handle" | "isPrivate" | "permissions"> & {
    __typename: "Project";
    labels?: ({ id: string } | LabelShape)[];
    owner: OwnerShape | null | undefined;
    parent?: { id: string } | null;
    tags?: ({ tag: string } | TagShape)[];
    versions?: Omit<ProjectVersionShape, "root">[] | null;
}

export const shapeProject: ShapeModel<ProjectShape, ProjectCreateInput, ProjectUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, "id", "handle", "isPrivate", "permissions"),
        ...createOwner(d, "ownedBy"),
        ...createRel(d, "labels", ["Connect", "Create"], "many", shapeLabel),
        ...createRel(d, "parent", ["Connect"], "one"),
        ...createRel(d, "tags", ["Connect", "Create"], "many", shapeTag),
        ...createVersion(d, shapeProjectVersion),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "handle", "isPrivate", "permissions"),
        ...updateOwner(o, u, "ownedBy"),
        ...updateRel(o, u, "labels", ["Connect", "Create", "Disconnect"], "many", shapeLabel),
        ...updateRel(o, u, "tags", ["Connect", "Create", "Disconnect"], "many", shapeTag),
        ...updateVersion(o, u, shapeProjectVersion),
    }, a),
};
