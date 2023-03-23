import { Project, ProjectCreateInput, ProjectUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { LabelShape, shapeLabel } from "./label";
import { ProjectVersionShape, shapeProjectVersion } from "./projectVersion";
import { shapeTag, TagShape } from "./tag";
import { createOwner, createPrims, createRel, createVersion, shapeUpdate, updateOwner, updatePrims, updateRel, updateVersion } from "./tools";
import { OwnerShape } from "./types";

export type ProjectShape = Pick<Project, 'id' | 'handle' | 'isPrivate' | 'permissions'> & {
    __typename?: 'Project';
    labels?: ({ id: string } | LabelShape)[];
    owner: OwnerShape | null;
    parent?: { id: string } | null;
    tags?: ({ tag: string } | TagShape)[];
    // Updating, deleting, and reordering versions must be done separately. 
    // This only creates/updates a single version, which is most often the case
    versionInfo?: ProjectVersionShape | null;
}

export const shapeProject: ShapeModel<ProjectShape, ProjectCreateInput, ProjectUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'handle', 'isPrivate', 'permissions'),
        ...createOwner(d, 'ownedBy'),
        ...createRel(d, 'labels', ['Connect', 'Create'], 'many', shapeLabel),
        ...createRel(d, 'parent', ['Connect'], 'one'),
        ...createRel(d, 'tags', ['Connect', 'Create'], 'many', shapeTag),
        ...createVersion(d, shapeProjectVersion, (v) => ({ ...v, root: { id: d.id } })),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'handle', 'isPrivate', 'permissions'),
        ...updateOwner(o, u, 'ownedBy'),
        ...updateRel(o, u, 'labels', ['Connect', 'Create', 'Disconnect'], 'many', shapeLabel),
        ...updateRel(o, u, 'tags', ['Connect', 'Create', 'Disconnect'], 'many', shapeTag),
        ...updateVersion(o, u, shapeProjectVersion, (v, i) => ({ ...v, root: { id: i.id } })),
    }, a)
}