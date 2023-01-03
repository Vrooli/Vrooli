import { Project, ProjectCreateInput, ProjectUpdateInput, ProjectVersionTranslation, ProjectVersionTranslationCreateInput, ProjectVersionTranslationUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { createPrims, ResourceListShape, shapeTagCreate, shapeUpdate, TagShape, updatePrims } from "utils";

export type ProjectVersionTranslationShape = Pick<ProjectVersionTranslation, 'id' | 'language' | 'description' | 'name'>

export type ProjectShape = Omit<OmitCalculated<Project>, 'resourceLists' | 'tags' | 'translations' | 'owner'> & {
    id: string;
    // handle: string | null; TODO
    resourceLists?: ResourceListShape[] | null;
    tags?: TagShape[];
    translations: ProjectVersionTranslationShape[];
    parent?: {
        id: string
    } | null;
    owner?: {
        __typename: 'User' | 'Organization';
        id: string;
    } | null;
}

export const shapeProjectVersionTranslation: ShapeModel<ProjectVersionTranslationShape, ProjectVersionTranslationCreateInput, ProjectVersionTranslationUpdateInput> = {
    create: (item) => createPrims(item, 'id', 'language', 'description', 'name'),
    update: (o, u) => shapeUpdate(u, updatePrims(o, u, 'id', 'description', 'name'))
}

export const shapeProject: ShapeModel<ProjectShape, ProjectCreateInput, ProjectUpdateInput> = {
    create: (item) => ({
        id: item.id,
        isComplete: item.isComplete,
        isPrivate: item.isPrivate,
        parentId: item.parent?.id,
        createdByUserId: item.owner?.__typename === 'User' ? item.owner.id : undefined,
        createdByOrganizationId: item.owner?.__typename === 'Organization' ? item.owner.id : undefined,
        ...shapeCreateList(item, 'translations', shapeProjectTranslation),
        ...shapeCreateList(item, 'resourceLists', shapeResourceList),
        ...shapeCreateList(item, 'tags', shapeTagCreate),
    }),
    update: shapeUpdate(u, {
        ...shapeUpdatePrims(o, u, 'id', 'isComplete', 'isPrivate'),
        userId: u.owner?.__typename === 'User' ? u.owner.id : undefined,
        organizationId: u.owner?.__typename === 'Organization' ? u.owner.id : undefined,
        ...shapeUpdateList(o, u, 'translations', shapeProjectVersionTranslation, 'id'),
        ...shapeUpdateList(o, u, 'resourceLists', shapeResourceList, 'id'),
        ...shapeUpdateList(o, u, 'tags', shapeTag, 'tag', true),
    })
}