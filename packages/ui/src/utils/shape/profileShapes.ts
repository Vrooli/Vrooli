import { ProfileUpdateInput, TagHiddenCreateInput, TagHiddenUpdateInput, UserTranslationCreateInput, UserTranslationUpdateInput } from "graphql/generated/globalTypes";
import { Profile, ProfileTranslation, ShapeWrapper, TagHidden } from "types";
import { hasObjectChanged } from "./objectTools";
import { ResourceListShape, shapeResourceListsUpdate } from "./resourceShapes";
import { shapeCreateList, shapeUpdate, shapeUpdateList } from "./shapeTools";
import { shapeTagCreate, shapeTagsCreate, shapeTagsUpdate, TagShape } from "./tagShapes";

export type ProfileTranslationShape = Omit<ShapeWrapper<ProfileTranslation>, 'language' | 'bio'> & {
    id: string;
    language: UserTranslationCreateInput['language'];
    bio: UserTranslationCreateInput['bio'];
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

export const shapeProfileTranslationsCreate = (items: ProfileTranslationShape[] | null | undefined): {
    translationsCreate?: UserTranslationCreateInput[],
} => shapeCreateList(items, 'translations', shapeProfileTranslationCreate);

export const shapeProfileTranslationsUpdate = (
    o: ProfileTranslationShape[] | null | undefined,
    u: ProfileTranslationShape[] | null | undefined
): {
    translationsCreate?: UserTranslationCreateInput[],
    translationsUpdate?: UserTranslationUpdateInput[],
    translationsDelete?: string[],
} => shapeUpdateList(o, u, 'translations', hasObjectChanged, shapeProfileTranslationCreate, shapeProfileTranslationUpdate)

export type TagHiddenShape = Omit<ShapeWrapper<TagHidden>, 'tags'> & {
    id: string;
    tag: TagShape;
}

export const shapeTagHiddenCreate = (item: TagHiddenShape): TagHiddenCreateInput => ({
    id: item.id,
    isBlur: item.isBlur,
    tagCreate: shapeTagCreate(item.tag),
})

export const shapeTagHiddenUpdate = (
    original: TagHiddenShape,
    updated: TagHiddenShape
): TagHiddenUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: u.id,
        isBlur: u.isBlur !== o.isBlur ? u.isBlur : undefined,
    }))

export const shapeTagHiddensUpdate = (
    o: TagHiddenShape[] | null | undefined,
    u: TagHiddenShape[] | null | undefined
): {
    hiddenTagsCreate?: TagHiddenCreateInput[],
    hiddenTagsUpdate?: TagHiddenUpdateInput[],
    hiddenTagsDelete?: string[],
} => shapeUpdateList(o, u, 'hiddenTags', hasObjectChanged, shapeTagHiddenCreate, shapeTagHiddenUpdate)

export type ProfileShape = Omit<ShapeWrapper<Profile>, 'resourceLists' | 'hiddenTags' | 'translations' | 'owner'> & {
    id: string;
    resourceLists?: ResourceListShape[] | null;
    hiddenTags?: TagHiddenShape[];
    translations: ProfileTranslationShape[];
}

export const shapeProfileUpdate = (
    original: ProfileShape,
    updated: ProfileShape
): ProfileUpdateInput | undefined =>
    shapeUpdate(original, updated, (o, u) => ({
        id: o.id,
        handle: updated.handle !== original.handle ? updated.handle : undefined,
        name: updated.name !== original.name ? updated.name : undefined,
        theme: updated.theme !== original.theme ? updated.theme : undefined,
        ...shapeTagHiddensUpdate(o.hiddenTags, u.hiddenTags),
        ...shapeProfileTranslationsUpdate(o.translations, u.translations),
        ...shapeResourceListsUpdate(o.resourceLists, u.resourceLists),
    }))