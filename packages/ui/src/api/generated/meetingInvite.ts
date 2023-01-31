import gql from 'graphql-tag';

export const meetingInviteFindOne = gql`fragment Label_full on Label {
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


query meetingInvite($input: FindByIdInput!) {
  meetingInvite(input: $input) {
    meeting {
        attendees {
            id
            name
            handle
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
}`;

export const meetingInviteFindMany = gql`
query meetingInvites($input: MeetingInviteSearchInput!) {
  meetingInvites(input: $input) {
    edges {
        cursor
        node {
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
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const meetingInviteCreate = gql`fragment Label_full on Label {
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


mutation meetingInviteCreate($input: MeetingInviteCreateInput!) {
  meetingInviteCreate(input: $input) {
    meeting {
        attendees {
            id
            name
            handle
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
}`;

export const meetingInviteUpdate = gql`fragment Label_full on Label {
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


mutation meetingInviteUpdate($input: MeetingInviteUpdateInput!) {
  meetingInviteUpdate(input: $input) {
    meeting {
        attendees {
            id
            name
            handle
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
}`;

export const meetingInviteAccept = gql`fragment Label_full on Label {
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


mutation meetingInviteAccept($input: FindByIdInput!) {
  meetingInviteAccept(input: $input) {
    meeting {
        attendees {
            id
            name
            handle
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
}`;

export const meetingInviteDecline = gql`fragment Label_full on Label {
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


mutation meetingInviteDecline($input: FindByIdInput!) {
  meetingInviteDecline(input: $input) {
    meeting {
        attendees {
            id
            name
            handle
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
}`;

