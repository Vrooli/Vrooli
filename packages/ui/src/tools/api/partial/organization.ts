import { Organization, OrganizationTranslation, OrganizationYou } from "@local/shared";
import { GqlPartial } from "../types";
import { rel } from "../utils";

export const organizationTranslation: GqlPartial<OrganizationTranslation> = {
    __typename: "OrganizationTranslation",
    common: {
        id: true,
        language: true,
        bio: true,
        name: true,
    },
    full: {},
    list: {},
};

export const organizationYou: GqlPartial<OrganizationYou> = {
    __typename: "OrganizationYou",
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

export const organization: GqlPartial<Organization> = {
    __typename: "Organization",
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
        translations: () => rel(organizationTranslation, "full"),
        you: () => rel(organizationYou, "full"),
    },
    full: {
        roles: async () => rel((await import("./role")).role, "full", { omit: "organization" }),
    },
    list: {},
    nav: {
        id: true,
        bannerImage: true,
        handle: true,
        profileImage: true,
        you: () => rel(organizationYou, "full"),
    },
};
