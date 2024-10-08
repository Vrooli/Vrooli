import { Project, ProjectCreateInput, ProjectUpdateInput } from "../../api/generated/graphqlTypes";
import { CanConnect, ShapeModel } from "../../consts/commonTypes";
import { LabelShape, shapeLabel } from "./label";
import { ProjectVersionShape, shapeProjectVersion } from "./projectVersion";
import { TagShape, shapeTag } from "./tag";
import { createOwner, createPrims, createRel, createVersion, shapeUpdate, updateOwner, updatePrims, updateRel, updateVersion } from "./tools";
import { OwnerShape } from "./types";

export type ProjectShape = Pick<Project, "id" | "handle" | "isPrivate" | "permissions"> & {
    __typename: "Project";
    labels?: CanConnect<LabelShape>[] | null;
    owner: OwnerShape | null | undefined;
    parent?: CanConnect<ProjectVersionShape> | null;
    tags?: CanConnect<TagShape, "tag">[] | null;
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
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, "id", "handle", "isPrivate", "permissions"),
        ...updateOwner(o, u, "ownedBy"),
        ...updateRel(o, u, "labels", ["Connect", "Create", "Disconnect"], "many", shapeLabel),
        ...updateRel(o, u, "tags", ["Connect", "Create", "Disconnect"], "many", shapeTag),
        ...updateVersion(o, u, shapeProjectVersion),
    }),
};
