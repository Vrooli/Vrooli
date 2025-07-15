import { type Team, type TeamTranslation, type TeamYou } from "@vrooli/shared";
import { type ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const teamTranslation: ApiPartial<TeamTranslation> = {
    common: {
        id: true,
        language: true,
        bio: true,
        name: true,
    },
};

export const teamYou: ApiPartial<TeamYou> = {
    common: {
        canAddMembers: true,
        canDelete: true,
        canBookmark: true,
        canReport: true,
        canUpdate: true,
        canRead: true,
        isBookmarked: true,
        isViewed: true,
        yourMembership: {
            id: true,
            createdAt: true,
            updatedAt: true,
            isAdmin: true,
            permissions: true,
        },
    },
};

export const team: ApiPartial<Team> = {
    common: {
        id: true,
        publicId: true,
        bannerImage: true,
        handle: true,
        createdAt: true,
        updatedAt: true,
        isOpenToNewMembers: true,
        isPrivate: true,
        commentsCount: true,
        membersCount: true,
        profileImage: true,
        reportsCount: true,
        bookmarks: true,
        tags: async () => rel((await import("./tag.js")).tag, "list"),
        translations: () => rel(teamTranslation, "full"),
        you: () => rel(teamYou, "full"),
    },
    full: {
        // @ts-expect-error - JSONB field - select entire JSON object
        config: true,
        members: async () => rel((await import("./member.js")).member, "list", { omit: "team" }),
    },
    nav: {
        id: true,
        bannerImage: true,
        handle: true,
        profileImage: true,
        you: () => rel(teamYou, "full"),
    },
};
