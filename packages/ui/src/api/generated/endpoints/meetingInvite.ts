import gql from 'graphql-tag';
import { Label_full } from '../fragments/Label_full';
import { Label_list } from '../fragments/Label_list';

export const meetingInviteFindOne = gql`${Label_full}

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

export const meetingInviteFindMany = gql`${Label_list}

query meetingInvites($input: MeetingInviteSearchInput!) {
  meetingInvites(input: $input) {
    edges {
        cursor
        node {
            meeting {
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

export const meetingInviteCreate = gql`${Label_full}

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

export const meetingInviteUpdate = gql`${Label_full}

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

export const meetingInviteAccept = gql`${Label_full}

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

export const meetingInviteDecline = gql`${Label_full}

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

