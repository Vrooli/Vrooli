import { Api, ApiCreateInput, ApiUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { createOwner, createPrims, createRel, LabelShape, shapeApiVersion, shapeLabel, shapeTag, shapeUpdate, TagShape, updateOwner, updatePrims, updateRel } from "utils";

export type ApiShape = Pick<Api, 'id'> & {
    labels?: LabelShape[];
    owner?: { __typename: 'User' | 'Organization', id: string } | null;
    parent?: { id: string } | null;
    tags?: TagShape[];
    // Updating, deleting, and reordering versions must be done separately. 
    // This only creates/updates a single version, which is most often the case
    versionLabel?: string | null;
    versionData?: ApiVersionShape | null;
}

export const shapeApi: ShapeModel<ApiShape, ApiCreateInput, ApiUpdateInput> = {
    create: (item) => ({
        ...createPrims(item, 'id'),
        ...createOwner(item),
        ...createRel(item, 'labels', ['Connect', 'Create'], 'many', shapeLabel),
        ...createRel(item, 'parent', ['Connect'], 'one'),
        ...createRel(item, 'tags', ['Connect', 'Create'], 'many', shapeTag, 'tag'),
        ...createVersion(item, shapeApiVersion),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id'),
        ...updateOwner(o, u),
        ...updateRel(o, u, 'labels', ['Connect', 'Create', 'Disconnect'], 'many', shapeLabel),
        ...updateRel(o, u, 'tags', ['Connect', 'Create', 'Disconnect'], 'many', shapeTag, 'tag'),
        ...updateVersion(o, u, shapeApiVersion),
    })
}