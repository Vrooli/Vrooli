import gql from 'graphql-tag';

export const findOne = gql`fragment Label_full on Label {
    apisCount
    issuesCount
    meetingsCount
    notesCount
    projectsCount
    routinesCount
    runProjectSchedulesCount
    runRoutineSchedulesCount
    smartContractsCount
    standardsCount
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}

fragment Organization_nav on Organization {
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
}

fragment User_nav on User {
    id
    name
    handle
}


query meeting($input: FindByIdInput!) {
  meeting(input: $input) {
    attendees {
        id
        name
        handle
    }
    invites {
        id
        created_at
        updated_at
        message
        status
        you {
            canDelete
            canEdit
        }
    }
    labels {
        ...Label_full
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
        canEdit
        canInvite
    }
  }
}`;

export const findMany = gql`fragment Label_list on Label {
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}

fragment Organization_nav on Organization {
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
}

fragment User_nav on User {
    id
    name
    handle
}


query meetings($input: MeetingSearchInput!) {
  meetings(input: $input) {
    edges {
        cursor
        node {
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
                canEdit
                canInvite
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const create = gql`fragment Label_full on Label {
    apisCount
    issuesCount
    meetingsCount
    notesCount
    projectsCount
    routinesCount
    runProjectSchedulesCount
    runRoutineSchedulesCount
    smartContractsCount
    standardsCount
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}

fragment Organization_nav on Organization {
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
}

fragment User_nav on User {
    id
    name
    handle
}


mutation meetingCreate($input: MeetingCreateInput!) {
  meetingCreate(input: $input) {
    attendees {
        id
        name
        handle
    }
    invites {
        id
        created_at
        updated_at
        message
        status
        you {
            canDelete
            canEdit
        }
    }
    labels {
        ...Label_full
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
        canEdit
        canInvite
    }
  }
}`;

export const update = gql`fragment Label_full on Label {
    apisCount
    issuesCount
    meetingsCount
    notesCount
    projectsCount
    routinesCount
    runProjectSchedulesCount
    runRoutineSchedulesCount
    smartContractsCount
    standardsCount
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}

fragment Organization_nav on Organization {
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
}

fragment User_nav on User {
    id
    name
    handle
}


mutation meetingUpdate($input: MeetingUpdateInput!) {
  meetingUpdate(input: $input) {
    attendees {
        id
        name
        handle
    }
    invites {
        id
        created_at
        updated_at
        message
        status
        you {
            canDelete
            canEdit
        }
    }
    labels {
        ...Label_full
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
        canEdit
        canInvite
    }
  }
}`;

