import gql from 'graphql-tag';
import { Label_full } from '../fragments/Label_full';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';

export const meetingInviteFindOne = gql`...${Label_full}
...${Organization_nav}
...${User_nav}

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

export const meetingInviteCreate = gql`...${Label_full}
...${Organization_nav}
...${User_nav}

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

export const meetingInviteUpdate = gql`...${Label_full}
...${Organization_nav}
...${User_nav}

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

export const meetingInviteAccept = gql`...${Label_full}
...${Organization_nav}
...${User_nav}

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

export const meetingInviteDecline = gql`...${Label_full}
...${Organization_nav}
...${User_nav}

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

