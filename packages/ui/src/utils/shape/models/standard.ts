import { Standard, StandardCreateInput, StandardUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { LabelShape, shapeLabel } from "./label";
import { shapeStandardVersion, StandardVersionShape } from "./standardVersion";
import { shapeTag, TagShape } from "./tag";
import { createOwner, createPrims, createRel, createVersion, shapeUpdate, updateOwner, updatePrims, updateRel, updateVersion } from "./tools";
import { OwnerShape } from "./types";


export type StandardShape = Pick<Standard, 'id' | 'isInternal' | 'isPrivate' | 'permissions'> & {
    __typename?: 'Standard';
    parent?: { id: string } | null;
    owner?: OwnerShape | null;
    labels?: ({ id: string } | LabelShape)[] | null;
    tags?: ({ tag: string } | TagShape)[] | null;
    // Updating, deleting, and reordering versions must be done separately. 
    // This only creates/updates a single version, which is most often the case
    versionInfo?: StandardVersionShape | null;
}

export type StandardShapeUpdate = Omit<StandardShape, 'default' | 'isInternal' | 'name' | 'props' | 'yup' | 'type' | 'version' | 'creator'>;

export const shapeStandard: ShapeModel<StandardShape, StandardCreateInput, StandardUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'isInternal', 'isPrivate', 'permissions'),
        ...createOwner(d, 'ownedBy'),
        ...createRel(d, 'labels', ['Connect', 'Create'], 'many', shapeLabel),
        ...createRel(d, 'parent', ['Connect'], 'one'),
        ...createRel(d, 'tags', ['Connect', 'Create'], 'many', shapeTag),
        ...createVersion(d, shapeStandardVersion, (v) => ({ ...v, root: { id: d.id } })),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'isInternal', 'isPrivate', 'permissions'),
        ...updateOwner(o, u, 'ownedBy'),
        ...updateRel(o, u, 'labels', ['Connect', 'Create', 'Disconnect'], 'many', shapeLabel),
        ...updateRel(o, u, 'tags', ['Connect', 'Create', 'Disconnect'], 'many', shapeTag),
        ...updateVersion(o, u, shapeStandardVersion, (v, i) => ({ ...v, root: { id: i.id } })),
    }, a)
}