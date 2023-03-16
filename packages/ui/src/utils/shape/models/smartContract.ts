import { SmartContract, SmartContractCreateInput, SmartContractUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { OwnerShape } from "./common";
import { LabelShape, shapeLabel } from "./label";
import { shapeSmartContractVersion, SmartContractVersionShape } from "./smartContractVersion";
import { shapeTag, TagShape } from "./tag";
import { createOwner, createPrims, createRel, createVersion, shapeUpdate, updateOwner, updatePrims, updateRel, updateVersion } from "./tools";

export type SmartContractShape = Pick<SmartContract, 'id' | 'isPrivate'> & {
    __typename?: 'SmartContract';
    labels?: ({ id: string } | LabelShape)[];
    owner: OwnerShape | null;
    parent?: { id: string } | null;
    tags?: ({ tag: string } | TagShape)[];
    // Updating, deleting, and reordering versions must be done separately. 
    // This only creates/updates a single version, which is most often the case
    versionInfo?: SmartContractVersionShape | null;
}

export const shapeSmartContract: ShapeModel<SmartContractShape, SmartContractCreateInput, SmartContractUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id', 'isPrivate'),
        ...createOwner(d, 'ownedBy'),
        ...createRel(d, 'labels', ['Connect', 'Create'], 'many', shapeLabel),
        ...createRel(d, 'parent', ['Connect'], 'one'),
        ...createRel(d, 'tags', ['Connect', 'Create'], 'many', shapeTag),
        ...createVersion(d, shapeSmartContractVersion, (v) => ({ ...v, root: { id: d.id } })),
    }),
    update: (o, u, a) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'isPrivate'),
        ...updateOwner(o, u, 'ownedBy'),
        ...updateRel(o, u, 'labels', ['Connect', 'Create', 'Disconnect'], 'many', shapeLabel),
        ...updateRel(o, u, 'tags', ['Connect', 'Create', 'Disconnect'], 'many', shapeTag),
        ...updateVersion(o, u, shapeSmartContractVersion, (v, i) => ({ ...v, root: { id: i.id } })),
    }, a)
}