import { Tag, TagTranslation, TagYou } from "@shared/consts";
import { relPartial } from "api/utils";
import { GqlPartial } from "types";

export const tagTranslationPartial: GqlPartial<TagTranslation> = {
    __typename: 'TagTranslation',
    full: {
        id: true,
        language: true,
        description: true,
    }
}

export const tagYouPartial: GqlPartial<TagYou> = {
    __typename: 'TagYou',
    full: {
        isOwn: true,
        isStarred: true,
    },
}

export const tagPartial: GqlPartial<Tag> = {
    __typename: 'Tag',
    full: {
        id: true,
        created_at: true,
        tag: true,
        stars: true,
        translations: () => relPartial(tagTranslationPartial, 'full'),
        you: () => relPartial(tagYouPartial, 'full'),
    },
}