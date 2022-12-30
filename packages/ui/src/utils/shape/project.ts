import { Project, ProjectCreateInput, ProjectUpdateInput, ProjectVersionTranslation } from "@shared/consts";
import { OmitCalculated } from "types";
import { hasObjectChanged, ResourceListShape, shapeResourceListCreate, shapeResourceListUpdate, shapeTagCreate, shapeTagUpdate, TagShape } from "utils";
import { shapeCreateList, shapeCreatePrims, shapeUpdatePrims, shapeUpdate, shapeUpdateList } from "./shapeTools";

export type ProjectVersionTranslationShape = OmitCalculated<ProjectVersionTranslation>

export type ProjectShape = Omit<OmitCalculated<Project>, 'resourceLists' | 'tags' | 'translations' | 'owner'> & {
    id: string;
    // handle: string | null; TODO
    resourceLists?: ResourceListShape[] | null;
    tags?: TagShape[];
    translations: ProjectTranslationShape[];
    parent?: {
        id: string
    } | null;
    owner?: {
        __typename: 'User' | 'Organization';
        id: string;
    } | null;
}

export const shapeProjectVersionTranslationCreate = (item: ProjectVersionTranslationShape): ProjectVersionTranslationCreateInput =>
    shapeCreatePrims(item, 'id', 'language', 'description', 'name')

export const shapeProjectVersionTranslationUpdate = (o: ProjectVersionTranslationShape, u: ProjectVersionTranslationShape): ProjectTranslationUpdateInput | undefined =>
    shapeUpdate(u, shapeUpdatePrims(o, u, 'id', 'description', 'name'))

export const shapeProjectCreate = (item: ProjectShape): ProjectCreateInput => ({
    id: item.id,
    isComplete: item.isComplete,
    isPrivate: item.isPrivate,
    parentId: item.parent?.id,
    createdByUserId: item.owner?.__typename === 'User' ? item.owner.id : undefined,
    createdByOrganizationId: item.owner?.__typename === 'Organization' ? item.owner.id : undefined,
    ...shapeCreateList(item, 'translations', shapeProjectTranslationCreate),
    ...shapeCreateList(item, 'resourceLists', shapeResourceListCreate),
    ...shapeCreateList(item, 'tags', shapeTagCreate),
})

export const shapeProjectUpdate = (o: ProjectShape, u: ProjectShape): ProjectUpdateInput | undefined =>
    shapeUpdate(u, {
        ...shapeUpdatePrims(o, u, 'id', 'isComplete', 'isPrivate'),
        userId: u.owner?.__typename === 'User' ? u.owner.id : undefined,
        organizationId: u.owner?.__typename === 'Organization' ? u.owner.id : undefined,
        ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeProjectTranslationCreate, shapeProjectVersionTranslationUpdate, 'id'),
        ...shapeUpdateList(o, u, 'resourceLists', hasObjectChanged, shapeResourceListCreate, shapeResourceListUpdate, 'id'),
        ...shapeUpdateList(o, u, 'tags', hasObjectChanged, shapeTagCreate, shapeTagUpdate, 'tag', true),
    })