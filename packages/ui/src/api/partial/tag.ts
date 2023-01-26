import { Tag, TagTranslation, TagYou } from "@shared/consts";
import { relPartial } from "api/utils";
import { GqlPartial } from "types";

export const tagTranslationPartial: GqlPartial<TagTranslation> = {
    __typename: 'TagTranslation',
    common: {
        id: true,
        language: true,
        description: true,
    },
    full: {},
    list: {},
}

export const tagYouPartial: GqlPartial<TagYou> = {
    __typename: 'TagYou',
    common: {
        isOwn: true,
        isStarred: true,
    },
    full: {},
    list: {},
}

export const tagPartial: GqlPartial<Tag> = {
    __typename: 'Tag',
    common: {
        id: true,
        created_at: true,
        tag: true,
        stars: true,
        translations: () => relPartial(tagTranslationPartial, 'full'),
        you: () => relPartial(tagYouPartial, 'full'),
    },
    full: {},
    list: {},
}