import gql from 'graphql-tag';
import { Label_full } from '../fragments/Label_full';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';
import { Label_list } from '../fragments/Label_list';

export const meetingFindOne = gql`${Label_full}
${Organization_nav}
${User_nav}

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
            canUpdate
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
}`;

export const meetingFindMany = gql`${Label_list}
${Organization_nav}
${User_nav}

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
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const meetingCreate = gql`${Label_full}
${Organization_nav}
${User_nav}

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
            canUpdate
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
}`;

export const meetingUpdate = gql`${Label_full}
${Organization_nav}
${User_nav}

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
            canUpdate
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
}`;
