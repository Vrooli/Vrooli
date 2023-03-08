import gql from 'graphql-tag';
import { Label_full } from '../fragments/Label_full';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';

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

