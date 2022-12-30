import { Api, ApiCreateInput, ApiUpdateInput } from "@shared/consts";
import { OmitCalculated } from "types";
import { hasObjectChanged, LabelShape, shapeLabelCreate, shapeLabelUpdate, shapeTagCreate, shapeTagUpdate, TagShape } from "utils";
import { shapeCreateList, shapeUpdate, shapeUpdateList } from "./shapeTools";

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

export const shapeApiCreate = (item: ApiShape, threadId: string | null | undefined): ApiCreateInput => {
    return {
        id: item.id,
        parentConnect: item.parent?.id,
        organizationConnect: item.owner?.__typename === 'User' ? item.owner.id : undefined,
        userConnect: item.owner?.__typename === 'Organization' ? item.owner.id : undefined,
        ...shapeCreateList(item, 'labels', shapeLabelCreate),
        ...shapeCreateList(item, 'tags', shapeTagCreate),
        ...shapeVersionCreate(item, shapeApiVersionCreate),
    }
}

export const shapeApiUpdate = (o: ApiShape, u: ApiShape): ApiUpdateInput | undefined =>
    shapeUpdate(u, {
        id: o.id,
        userId: u.owner?.__typename === 'User' ? u.owner.id : undefined,
        organizationId: u.owner?.__typename === 'Organization' ? u.owner.id : undefined,
        ...shapeUpdateList(o, u, 'labels', hasObjectChanged, shapeLabelCreate, shapeLabelUpdate, 'id'),
        ...shapeUpdateList(o, u, 'tags', hasObjectChanged, shapeTagCreate, shapeTagUpdate, 'tag', true),
        ...shapeVersionUpdate(o, u, shapeApiVersionCreate, shapeApiVersionUpdate),
    })