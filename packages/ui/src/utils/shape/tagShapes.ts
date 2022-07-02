import { TagSelectorTag } from "components/inputs/types";

export type ShapeTagsCreateResult = {
    tagsCreate?: { tag: string }[];
    tagsConnect?: string[];
};

export type ShapeTagsUpdateResult = {
    tagsCreate?: { tag: string }[];
    tagsConnect?: string[];
    tagsDisconnect?: string[];
};

/**
 * Shapes a tag list associated with an object being created
 * @param tags The tags to shape
 * @returns An object shaped for the API
 */
export const shapeTagsCreate = (tags: TagSelectorTag[]): ShapeTagsCreateResult => {
    return Array.isArray(tags) && tags.length > 0 ? {
        tagsCreate: tags.filter(t => !t.id).map(t => ({ tag: t.tag })),
        tagsConnect: tags.filter(t => t.id).map(t => (t.id)) as string[],
    } : {};
}

/**
 * Shapes a tag list associated with an object being updated
 * @param existingTags Tags that already exist in the object
 * @param newTags Tags that are being added to the object
 * @returns An object shaped for the API
 */
export const shapeTagsUpdate = (existingTags: TagSelectorTag[] | null | undefined, newTags: TagSelectorTag[]): ShapeTagsUpdateResult => {
    console.log('SHAPE TAGS UPDATE', existingTags, newTags);
    if (!existingTags || !Array.isArray(existingTags)) {
        return shapeTagsCreate(newTags);
    }
    return Array.isArray(newTags) && newTags.length > 0 ? {
        tagsCreate: newTags.filter(t => !t.id && !existingTags.some(tag => tag.tag === t.tag)).map(t => ({ tag: t.tag })),
        tagsConnect: newTags.filter(t => t.id && !existingTags.some(tag => tag.tag === t.tag)).map(t => (t.id)) as string[],
        tagsDisconnect: existingTags.filter(t => !newTags.some(tag => tag.tag === t.tag)).map(t => (t.id)) as string[],
    } : {}
}