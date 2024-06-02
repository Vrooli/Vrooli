import { Team, TeamTranslation, TeamYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const teamTranslation: GqlPartial<TeamTranslation> = {
    __typename: "TeamTranslation",
    common: {
        id: true,
        language: true,
        bio: true,
        name: true,
    },
    full: {},
    list: {},
};

export const teamYou: GqlPartial<TeamYou> = {
    __typename: "TeamYou",
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
    full: {},
    list: {},
};

export const team: GqlPartial<Team> = {
    __typename: "Team",
    common: {
        __define: {
            0: async () => rel((await import("./tag")).tag, "list"),
        },
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
        tags: { __use: 0 },
        translations: () => rel(teamTranslation, "full"),
        you: () => rel(teamYou, "full"),
    },
    full: {
        roles: async () => rel((await import("./role")).role, "full", { omit: "team" }),
    },
    list: {},
    nav: {
        id: true,
        bannerImage: true,
        handle: true,
        profileImage: true,
        you: () => rel(teamYou, "full"),
    },
};
