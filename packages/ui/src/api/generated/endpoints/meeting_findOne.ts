import gql from 'graphql-tag';
import { Label_full } from '../fragments/Label_full';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';

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
  }
}`;

