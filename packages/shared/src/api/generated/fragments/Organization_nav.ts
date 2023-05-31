export const Organization_nav = `fragment Organization_nav on Organization {
id
handle
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
