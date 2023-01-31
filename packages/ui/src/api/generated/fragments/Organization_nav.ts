export const Organization_nav = `fragment Organization_nav on Organization {
id
handle
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