import { Tag, TagTranslation, TagYou } from "@local/shared";
import { ApiPartial } from "../types";
import { rel } from "../utils";

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
        created_at: true,
        tag: true,
        bookmarks: true,
        translations: () => rel(tagTranslation, "full"),
        you: () => rel(tagYou, "full"),
    },
};
