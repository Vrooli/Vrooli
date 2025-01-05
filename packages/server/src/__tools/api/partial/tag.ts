import { Tag, TagTranslation, TagYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const tagTranslation: GqlPartial<TagTranslation> = {
    common: {
        id: true,
        language: true,
        description: true,
    },
};

export const tagYou: GqlPartial<TagYou> = {
    common: {
        isOwn: true,
        isBookmarked: true,
    },
};

export const tag: GqlPartial<Tag> = {
    common: {
        id: true,
        created_at: true,
        tag: true,
        bookmarks: true,
        translations: () => rel(tagTranslation, "full"),
        you: () => rel(tagYou, "full"),
    },
};
