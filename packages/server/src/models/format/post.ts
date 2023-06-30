import { PostModelLogic } from "../base/types";
import { Formatter } from "../types";

const __typename = "Post" as const;
export const PostFormat: Formatter<PostModelLogic> = {
    gqlRelMap: {
        __typename,
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
        __typename,
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
