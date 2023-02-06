import gql from 'graphql-tag';
import { Label_full } from '../fragments/Label_full';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';
import { Label_list } from '../fragments/Label_list';

export const meetingInviteFindOne = gql`${Label_full}
${Organization_nav}
${User_nav}

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
    }
    id
    created_at
    updated_at
    message
    status
    you {
        canDelete
        canUpdate
    }
  }
}`;

export const meetingInviteFindMany = gql`${Label_list}
${Organization_nav}
${User_nav}

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
            }
            id
            created_at
            updated_at
            message
            status
            you {
                canDelete
                canUpdate
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
${Organization_nav}
${User_nav}

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
    }
    id
    created_at
    updated_at
    message
    status
    you {
        canDelete
        canUpdate
    }
  }
}`;

export const meetingInviteUpdate = gql`${Label_full}
${Organization_nav}
${User_nav}

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
    }
    id
    created_at
    updated_at
    message
    status
    you {
        canDelete
        canUpdate
    }
  }
}`;

export const meetingInviteAccept = gql`${Label_full}
${Organization_nav}
${User_nav}

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
    }
    id
    created_at
    updated_at
    message
    status
    you {
        canDelete
        canUpdate
    }
  }
}`;

export const meetingInviteDecline = gql`${Label_full}
${Organization_nav}
${User_nav}

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
    }
    id
    created_at
    updated_at
    message
    status
    you {
        canDelete
        canUpdate
    }
  }
}`;

