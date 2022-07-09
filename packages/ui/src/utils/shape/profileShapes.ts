import { ProfileUpdateInput, TagHiddenCreateInput, TagHiddenUpdateInput, UserTranslationCreateInput, UserTranslationUpdateInput } from "graphql/generated/globalTypes";
import { Profile, ProfileTranslation, ShapeWrapper, TagHidden } from "types";
import { hasObjectChanged } from "./objectTools";
import { ResourceListShape } from "./resourceShapes";
import { shapeUpdate, shapeUpdateList } from "./shapeTools";
import { shapeTagCreate, shapeTagsUpdate, TagShape } from "./tagShapes";

export type ProfileTranslationShape = Omit<ShapeWrapper<ProfileTranslation>, 'language' | 'bio'> & {
    id: string;
    language: UserTranslationCreateInput['language'];
    bio: UserTranslationCreateInput['bio'];
}

export type TagHiddenShape = Omit<ShapeWrapper<TagHidden>, 'id' | 'tag'> & {
    id: string;
    tag: TagShape;
}

export type TagHiddenShapeUpdate = Omit<TagHiddenShape, 'tag'>;

export type ProfileShape = Omit<ShapeWrapper<Profile>, 'resourceLists' | 'hiddenTags' | 'starredTags' | 'translations' | 'owner'> & {
    id: string;
    resourceLists?: ResourceListShape[] | null;
    hiddenTags?: TagHiddenShape[] | null;
    starredTags?: TagShape[] | null;
    translations?: ProfileTranslationShape[] | null;
}

export const shapeProfileTranslationCreate = (item: ProfileTranslationShape): UserTranslationCreateInput => ({
    id: item.id,
    language: item.language,
    bio: item.bio,
})

export const shapeProfileTranslationUpdate = (
    original: ProfileTranslationShape,
    updated: ProfileTranslationShape
): UserTranslationUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: u.id,
        bio: u.bio !== o.bio ? u.bio : undefined,
    }))

export const shapeTagHiddenCreate = (item: TagHiddenShape): TagHiddenCreateInput => ({
    id: item.id,
    isBlur: item.isBlur,
    tagCreate: shapeTagCreate(item.tag),
})

export const shapeTagHiddenUpdate = (
    original: TagHiddenShapeUpdate,
    updated: TagHiddenShapeUpdate
): TagHiddenUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: u.id,
        isBlur: u.isBlur !== o.isBlur ? u.isBlur : undefined,
    }))

export const shapeProfileUpdate = (
    original: ProfileShape,
    updated: ProfileShape
): ProfileUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: o.id,
        handle: updated.handle !== original.handle ? updated.handle : undefined,
        name: updated.name !== original.name ? updated.name : undefined,
        theme: updated.theme !== original.theme ? updated.theme : undefined,
        ...shapeUpdateList(o, u, 'hiddenTags', hasObjectChanged, shapeTagHiddenCreate, shapeTagHiddenUpdate),
        ...shapeUpdateList(o, u, 'starredTags', hasObjectChanged, shapeTagCreate, shapeTagsUpdate),
        ...shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeProfileTranslationCreate, shapeProfileTranslationUpdate),
        ...shapeUpdateList(o, u, 'resourceLists', hasObjectChanged, shapeResourceListCreate, shapeResourceListUpdate),
    }))