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
stars
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
    canEdit
    canStar
    canReport
    canView
    isStarred
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