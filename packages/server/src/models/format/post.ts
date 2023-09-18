import { PostModelLogic } from "../base/types";
import { Formatter } from "../types";

export const PostFormat: Formatter<PostModelLogic> = {
    gqlRelMap: {
        __typename: "Post",
        comments: "Comment",
        owner: {
            user: "User",
            organization: "Organization",
        },
        reports: "Report",
        repostedFrom: "Post",
        reposts: "Post",
        resourceList: "ResourceList",
        bookmarkedBy: "User",
        tags: "Tag",
    },
    prismaRelMap: {
        __typename: "Post",
        organization: "Organization",
        user: "User",
        repostedFrom: "Post",
        reposts: "Post",
        resourceList: "ResourceList",
        comments: "Comment",
        bookmarkedBy: "User",
        reactions: "Reaction",
        viewedBy: "View",
        reports: "Report",
        tags: "Tag",
    },
    countFields: {
        commentsCount: true,
        repostsCount: true,
    },
};
