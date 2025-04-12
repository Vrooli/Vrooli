import { Team, TeamTranslation, TeamYou } from "@local/shared";
import { ApiPartial } from "../types.js";
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
            created_at: true,
            updated_at: true,
            isAdmin: true,
            permissions: true,
        },
    },
};

export const team: ApiPartial<Team> = {
    common: {
        id: true,
        bannerImage: true,
        handle: true,
        created_at: true,
        updated_at: true,
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
        members: async () => rel((await import("./member.js")).member, "list", { omit: "team" }),
        roles: async () => rel((await import("./role.js")).role, "full", { omit: "team" }),
    },
    nav: {
        id: true,
        bannerImage: true,
        handle: true,
        profileImage: true,
        you: () => rel(teamYou, "full"),
    },
};
