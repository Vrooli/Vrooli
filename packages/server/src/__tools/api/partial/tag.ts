import { Tag, TagTranslation, TagYou } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const tagTranslation: ApiPartial<TagTranslation> = {
    common: {
        id: true,
        language: true,
        description: true,
    },
};

export const tagYou: ApiPartial<TagYou> = {
    common: {
        isOwn: true,
        isBookmarked: true,
    },
};

export const tag: ApiPartial<Tag> = {
    common: {
        id: true,
        createdAt: true,
        tag: true,
        bookmarks: true,
        translations: () => rel(tagTranslation, "full"),
        you: () => rel(tagYou, "full"),
    },
};
