import { Api, ApiCreateInput, ApiUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { ApiVersionShape, createOwner, createPrims, createRel, createVersion, LabelShape, shapeApiVersion, shapeLabel, shapeTag, shapeUpdate, TagShape, updateOwner, updatePrims, updateRel, updateVersion } from "utils";

export type ApiShape = Pick<Api, 'id'> & {
    labels?: ({ id: string } | LabelShape)[];
    owner?: { type: 'User' | 'Organization', id: string } | null;
    parent?: { id: string } | null;
    tags?: ({ tag: string } | TagShape)[];
    // Updating, deleting, and reordering versions must be done separately. 
    // This only creates/updates a single version, which is most often the case
    versionInfo?: ApiVersionShape | null;
}

export const shapeApi: ShapeModel<ApiShape, ApiCreateInput, ApiUpdateInput> = {
    create: (d) => ({
        ...createPrims(d, 'id'),
        ...createOwner(d),
        ...createRel(d, 'labels', ['Connect', 'Create'], 'many', shapeLabel),
        ...createRel(d, 'parent', ['Connect'], 'one'),
        ...createRel(d, 'tags', ['Connect', 'Create'], 'many', shapeTag),
        ...createVersion(d, shapeApiVersion, (v) => ({ ...v, root: { id: d.id } })),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id'),
        ...updateOwner(o, u),
        ...updateRel(o, u, 'labels', ['Connect', 'Create', 'Disconnect'], 'many', shapeLabel),
        ...updateRel(o, u, 'tags', ['Connect', 'Create', 'Disconnect'], 'many', shapeTag),
        ...updateVersion(o, u, shapeApiVersion, (v, i) => ({ ...v, root: { id: i.id } })),
    })
}