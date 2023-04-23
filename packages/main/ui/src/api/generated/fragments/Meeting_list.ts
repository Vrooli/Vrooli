export const Meeting_list = `fragment Meeting_list on Meeting {
labels {
    ...Label_list
}
schedule {
    ...Schedule_list
}
translations {
    id
    language
    description
    link
    name
}
id
openToAnyoneWithInvite
showOnOrganizationProfile
organization {
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
}
restrictedToRoles {
    members {
        id
        created_at
        updated_at
        isAdmin
        permissions
        roles {
            id
            created_at
            updated_at
            name
            permissions
            membersCount
            organization {
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
            }
            translations {
                id
                language
                description
            }
        }
    }
    id
    created_at
    updated_at
    name
    permissions
    membersCount
    organization {
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
    }
    translations {
        id
        language
        description
    }
}
attendeesCount
invitesCount
you {
    canDelete
    canInvite
    canUpdate
}
}`;
