import { Routine, RoutineCreateInput, RoutineUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { TagShape, createPrims, updatePrims, shapeUpdate, updateRel, createRel, shapeTag, updateOwner, createOwner, shapeLabel, LabelShape, RoutineVersionShape, shapeRoutineVersion, createVersion, updateVersion } from "utils";


export type RoutineShape = Pick<Routine, 'id' | 'isInternal' | 'isPrivate' | 'permissions'> & {
    labels?: ({ id: string } | LabelShape)[];
    owner?: { __typename: 'User' | 'Organization', id: string } | null;
    parent?: { id: string } | null;
    tags?: ({ tag: string } | TagShape)[];
    // Updating, deleting, and reordering versions must be done separately. 
    // This only creates/updates a single version, which is most often the case
    versionInfo?: RoutineVersionShape | null;
}

export const shapeRoutine: ShapeModel<RoutineShape, RoutineCreateInput, RoutineUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'isInternal', 'isPrivate', 'permissions'),
        ...createOwner(d),
        ...createRel(d, 'labels', ['Connect', 'Create'], 'many', shapeLabel),
        ...createRel(d, 'parent', ['Connect'], 'one'),
        ...createRel(d, 'tags', ['Connect', 'Create'], 'many', shapeTag),
        ...createVersion(d, shapeRoutineVersion, (v) => ({ ...v, root: { id: d.id } })),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'isInternal', 'isPrivate', 'permissions'),
        ...updateOwner(o, u),
        ...updateRel(o, u, 'labels', ['Connect', 'Create', 'Disconnect'], 'many', shapeLabel),
        ...updateRel(o, u, 'tags', ['Connect', 'Create', 'Disconnect'], 'many', shapeTag),
        ...updateVersion(o, u, shapeRoutineVersion, (v, i) => ({ ...v, root: { id: i.id } })),
    }, a)
}