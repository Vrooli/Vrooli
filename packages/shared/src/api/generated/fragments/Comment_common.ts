export const Comment_common = `fragment Comment_common on Comment {
id
created_at
updated_at
owner {
    ... on Organization {
        ...Organization_nav
    }
    ... on User {
        ...User_nav
    }
}
score
bookmarks
reportsCount
you {
    canDelete
    canBookmark
    canReply
    canReport
    canUpdate
    canReact
    isBookmarked
    reaction
}
}`;
