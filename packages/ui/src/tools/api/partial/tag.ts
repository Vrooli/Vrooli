import { Tag, TagTranslation, TagYou } from "@shared/consts";
import { rel } from "../utils";
import { GqlPartial } from "../types";

export const tagTranslation: GqlPartial<TagTranslation> = {
    __typename: 'TagTranslation',
    common: {
        id: true,
        language: true,
        description: true,
    },
    full: {},
    list: {},
}

export const tagYou: GqlPartial<TagYou> = {
    __typename: 'TagYou',
    common: {
        isOwn: true,
        isBookmarked: true,
    },
    full: {},
    list: {},
}

export const tag: GqlPartial<Tag> = {
    __typename: 'Tag',
    common: {
        id: true,
        created_at: true,
        tag: true,
        bookmarks: true,
        translations: () => rel(tagTranslation, 'full'),
        you: () => rel(tagYou, 'full'),
    },
    full: {},
    list: {},
}