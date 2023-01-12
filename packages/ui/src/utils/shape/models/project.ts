import { Project, ProjectCreateInput, ProjectUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { createOwner, createRel, ProjectVersionShape, LabelShape, shapeTag, shapeUpdate, TagShape, updateOwner, updatePrims, updateRel, shapeProjectVersion, shapeLabel, createVersion, updateVersion, createPrims } from "utils";

export type ProjectShape = Pick<Project, 'id' | 'handle' | 'isPrivate' | 'permissions'> & {
    labels?: ({ id: string } | LabelShape)[];
    owner?: { type: 'User' | 'Organization', id: string } | null;
    parent?: { id: string } | null;
    tags?: ({ tag: string } | TagShape)[];
    // Updating, deleting, and reordering versions must be done separately. 
    // This only creates/updates a single version, which is most often the case
    versionInfo?: ProjectVersionShape | null;
}

export const shapeProject: ShapeModel<ProjectShape, ProjectCreateInput, ProjectUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id'),
        ...createOwner(d),
        ...createRel(d, 'labels', ['Connect', 'Create'], 'many', shapeLabel),
        ...createRel(d, 'parent', ['Connect'], 'one'),
        ...createRel(d, 'tags', ['Connect', 'Create'], 'many', shapeTag),
        ...createVersion(d, shapeProjectVersion, (v) => ({ ...v, root: { id: d.id } })),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id'),
        ...updateOwner(o, u),
        ...updateRel(o, u, 'labels', ['Connect', 'Create', 'Disconnect'], 'many', shapeLabel),
        ...updateRel(o, u, 'tags', ['Connect', 'Create', 'Disconnect'], 'many', shapeTag),
        ...updateVersion(o, u, shapeProjectVersion, (v, i) => ({ ...v, root: { id: i.id } })),
    })
}