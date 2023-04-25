import { Tag, TagTranslation, TagYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

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