import { Organization, OrganizationCreateInput, OrganizationTranslation, OrganizationTranslationCreateInput, OrganizationTranslationUpdateInput, OrganizationUpdateInput } from "@shared/consts";
import { ShapeModel } from "types";
import { createPrims, hasObjectChanged, ResourceListShape, shapeResourceListCreate, shapeResourceListUpdate, shapeTagCreate, shapeTagUpdate, shapeUpdate, TagShape, updatePrims } from "utils";

export type OrganizationTranslationShape = Pick<OrganizationTranslation, 'id' | 'language' | 'bio' | 'name'>

export type OrganizationShape = Omit<OmitCalculated<Organization>, 'handle' | 'isOpenToNewMembers' | 'resourceLists' | 'resourceLists' | 'tags' | 'translations'> & {
    id: string;
    handle: OrganizationCreateInput['handle'];
    isOpenToNewMembers: OrganizationCreateInput['isOpenToNewMembers'];
    resourceLists?: ResourceListShape[];
    tags?: TagShape[];
    translations: OrganizationTranslationShape[];
}

export const shapeOrganizationTranslation: ShapeModel<OrganizationTranslationShape, OrganizationTranslationCreateInput, OrganizationTranslationUpdateInput> = {
    create: (item) => createPrims(item, 'id', 'language', 'bio', 'name'),
    update: (o, u) => shapeUpdate(u, updatePrims(o, u, 'id', 'bio', 'name'))
}

export const shapeOrganization: ShapeModel<OrganizationShape, OrganizationCreateInput, OrganizationUpdateInput> = {
    create: (item) => ({
        ...createPrims(item, 'id', 'handle', 'isOpenToNewMembers', 'isPrivate'),
        ...shapeCreateList(item, 'translations', shapeOrganizationTranslationCreate),
        ...shapeCreateList(item, 'resourceLists', shapeResourceListCreate),
        ...shapeCreateList(item, 'tags', shapeTagCreate),
    }),
    update: (o, u) => shapeUpdate(u, {
        ...updatePrims(o, u, 'id', 'handle', 'isOpenToNewMembers', 'isPrivate'),
        ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeOrganizationTranslationCreate, shapeOrganizationTranslationUpdate, 'id'),
        ...shapeUpdateList(o, u, 'resourceLists', hasObjectChanged, shapeResourceListCreate, shapeResourceListUpdate, 'id'),
        ...shapeUpdateList(o, u, 'tags', hasObjectChanged, shapeTagCreate, shapeTagUpdate, 'tag', true, true),
        // TODO members
    })
}