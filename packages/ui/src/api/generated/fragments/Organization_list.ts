export const Organization_list = `fragment Organization_list on Organization {
id
handle
created_at
updated_at
isOpenToNewMembers
isPrivate
commentsCount
membersCount
reportsCount
bookmarks
tags {
    ...Tag_list
}
translations {
    id
    language
    bio
    name
}
you {
    canAddMembers
    canDelete
    canBookmark
    canReport
    canUpdate
    canRead
    isBookmarked
    isViewed
    yourMembership {
        id
        created_at
        updated_at
        isAdmin
        permissions
    }
}
}`;