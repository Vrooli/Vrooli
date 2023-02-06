export const Organization_nav = `fragment Organization_nav on Organization {
id
handle
you {
    canAddMembers
    canDelete
    canStar
    canReport
    canUpdate
    canRead
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