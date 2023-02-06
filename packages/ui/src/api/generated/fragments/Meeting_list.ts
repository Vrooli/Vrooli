export const Meeting_list = `fragment Meeting_list on Meeting {
labels {
    ...Label_list
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
timeZone
eventStart
eventEnd
recurring
recurrStart
recurrEnd
organization {
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
}
restrictedToRoles {
    members {
        id
        created_at
        updated_at
        isAdmin
        permissions
        organization {
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
        }
        user {
            id
            name
            handle
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