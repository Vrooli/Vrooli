import { Standard, StandardCreateInput, StandardUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { shapeUpdate, updatePrims, TagShape, createRel, updateRel, shapeTag, createOwner, updateOwner, LabelShape, StandardVersionShape, shapeStandardVersion, shapeLabel, createPrims, createVersion, updateVersion } from "utils";


export type StandardShape = Pick<Standard, 'id' | 'name' | 'isInternal' | 'isPrivate' | 'permissions'> & {
    parent?: { id: string };
    owner?: { type: 'User' | 'Organization', id: string } | null;
    labels?: ({ id: string } | LabelShape)[];
    tags?: ({ tag: string } | TagShape)[];
    // Updating, deleting, and reordering versions must be done separately. 
    // This only creates/updates a single version, which is most often the case
    versionInfo?: StandardVersionShape | null;
}

export type StandardShapeUpdate = Omit<StandardShape, 'default' | 'isInternal' | 'name' | 'props' | 'yup' | 'type' | 'version' | 'creator'>;

export const shapeStandard: ShapeModel<StandardShape, StandardCreateInput, StandardUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'name', 'isInternal', 'isPrivate', 'permissions'),
        ...createOwner(d),
        ...createRel(d, 'labels', ['Connect', 'Create'], 'many', shapeLabel),
        ...createRel(d, 'parent', ['Connect'], 'one'),
        ...createRel(d, 'tags', ['Connect', 'Create'], 'many', shapeTag),
        ...createVersion(d, shapeStandardVersion, (v) => ({ ...v, root: { id: d.id } })),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'name', 'isInternal', 'isPrivate', 'permissions'),
        ...updateOwner(o, u),
        ...updateRel(o, u, 'labels', ['Connect', 'Create', 'Disconnect'], 'many', shapeLabel),
        ...updateRel(o, u, 'tags', ['Connect', 'Create', 'Disconnect'], 'many', shapeTag),
        ...updateVersion(o, u, shapeStandardVersion, (v, i) => ({ ...v, root: { id: i.id } })),
    })
}