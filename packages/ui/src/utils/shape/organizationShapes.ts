import { OrganizationCreateInput, OrganizationTranslationCreateInput, OrganizationTranslationUpdateInput, OrganizationUpdateInput } from "graphql/generated/globalTypes";
import { ShapeWrapper, Organization, OrganizationTranslation } from "types";
import { hasObjectChanged, ResourceListShape, shapeResourceListCreate, shapeResourceListUpdate, shapeTagCreate, shapeTagUpdate, TagShape } from "utils";
import { shapeCreateList, shapeUpdate, shapeUpdateList } from "./shapeTools";

export type OrganizationTranslationShape = Omit<ShapeWrapper<OrganizationTranslation>, 'language' | 'name'> & {
    id: string;
    language: OrganizationTranslationCreateInput['language'];
    name: OrganizationTranslationCreateInput['name'];
}

export type OrganizationShape = Omit<ShapeWrapper<Organization>, 'handle' | 'isOpenToNewMembers' | 'resourceLists' | 'resourceLists' | 'tags' | 'translations'> & {
    id: string;
    // handle: OrganizationCreateInput['handle'];
    isOpenToNewMembers: OrganizationCreateInput['isOpenToNewMembers'];
    resourceLists?: ResourceListShape[];
    tags?: TagShape[];
    translations: OrganizationTranslationShape[];
}

export const shapeOrganizationTranslationCreate = (item: OrganizationTranslationShape): OrganizationTranslationCreateInput => ({
    id: item.id,
    language: item.language,
    name: item.name,
    bio: item.bio,
})

export const shapeOrganizationTranslationUpdate = (
    original: OrganizationTranslationShape,
    updated: OrganizationTranslationShape
): OrganizationTranslationUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: u.id,
        name: u.name !== o.name ? u.name : undefined,
        bio: u.bio !== o.bio ? u.bio : undefined,
    }), 'id')

export const shapeOrganizationCreate = (item: OrganizationShape): OrganizationCreateInput => ({
    id: item.id,
    isOpenToNewMembers: item.isOpenToNewMembers,
    ...shapeCreateList(item, 'translations', shapeOrganizationTranslationCreate),
    ...shapeCreateList(item, 'resourceLists', shapeResourceListCreate),
    ...shapeCreateList(item, 'tags', shapeTagCreate),
    //TODO handle
})

export const shapeOrganizationUpdate = (
    original: OrganizationShape,
    updated: OrganizationShape
): OrganizationUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: o.id,
        isOpenToNewMembers: u.isOpenToNewMembers !== o.isOpenToNewMembers ? u.isOpenToNewMembers : undefined,
        ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeOrganizationTranslationCreate, shapeOrganizationTranslationUpdate, 'id'),
        ...shapeUpdateList(o, u, 'resourceLists', hasObjectChanged, shapeResourceListCreate, shapeResourceListUpdate, 'id'),
        ...shapeUpdateList(o, u, 'tags', hasObjectChanged, shapeTagCreate, shapeTagUpdate, 'tag', true, true),
        // TODO members
        //TODO handl
    }), 'id')