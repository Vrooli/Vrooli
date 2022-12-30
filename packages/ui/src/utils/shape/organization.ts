import { Organization, OrganizationCreateInput, OrganizationTranslation, OrganizationTranslationCreateInput, OrganizationTranslationUpdateInput, OrganizationUpdateInput } from "@shared/consts";
import { OmitCalculated } from "types";
import { hasObjectChanged, ResourceListShape, shapeResourceListCreate, shapeResourceListUpdate, shapeTagCreate, shapeTagUpdate, TagShape } from "utils";
import { shapeCreateList, shapeCreatePrims, shapeUpdatePrims, shapeUpdate, shapeUpdateList } from "./shapeTools";

export type OrganizationTranslationShape = OmitCalculated<OrganizationTranslation>

export type OrganizationShape = Omit<OmitCalculated<Organization>, 'handle' | 'isOpenToNewMembers' | 'resourceLists' | 'resourceLists' | 'tags' | 'translations'> & {
    id: string;
    handle: OrganizationCreateInput['handle'];
    isOpenToNewMembers: OrganizationCreateInput['isOpenToNewMembers'];
    resourceLists?: ResourceListShape[];
    tags?: TagShape[];
    translations: OrganizationTranslationShape[];
}

export const shapeOrganizationTranslationCreate = (item: OrganizationTranslationShape): OrganizationTranslationCreateInput =>
    shapeCreatePrims(item, 'id', 'language', 'bio', 'name')

export const shapeOrganizationTranslationUpdate = (o: OrganizationTranslationShape, u: OrganizationTranslationShape): OrganizationTranslationUpdateInput | undefined =>
    shapeUpdate(u, shapeUpdatePrims(o, u, 'id', 'bio', 'name'))

export const shapeOrganizationCreate = (item: OrganizationShape): OrganizationCreateInput => ({
    ...shapeCreatePrims(item, 'id', 'handle', 'isOpenToNewMembers', 'isPrivate'),
    ...shapeCreateList(item, 'translations', shapeOrganizationTranslationCreate),
    ...shapeCreateList(item, 'resourceLists', shapeResourceListCreate),
    ...shapeCreateList(item, 'tags', shapeTagCreate),
})

export const shapeOrganizationUpdate = (o: OrganizationShape, u: OrganizationShape): OrganizationUpdateInput | undefined =>
    shapeUpdate(u, {
        ...shapeUpdatePrims(o, u, 'id', 'handle', 'isOpenToNewMembers', 'isPrivate'),
        ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeOrganizationTranslationCreate, shapeOrganizationTranslationUpdate, 'id'),
        ...shapeUpdateList(o, u, 'resourceLists', hasObjectChanged, shapeResourceListCreate, shapeResourceListUpdate, 'id'),
        ...shapeUpdateList(o, u, 'tags', hasObjectChanged, shapeTagCreate, shapeTagUpdate, 'tag', true, true),
        // TODO members
    })