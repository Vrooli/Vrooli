export const Comment_list = `fragment Comment_list on Comment {
translations {
    id
    language
    text
}
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
stars
reportsCount
you {
    canDelete
    canStar
    canReply
    canReport
    canUpdate
    canVote
    isStarred
    isUpvoted
}
}`;