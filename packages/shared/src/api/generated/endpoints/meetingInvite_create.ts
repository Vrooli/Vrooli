import gql from "graphql-tag";
import { Label_full } from "../fragments/Label_full";
import { Organization_nav } from "../fragments/Organization_nav";
import { Schedule_full } from "../fragments/Schedule_full";
import { User_nav } from "../fragments/User_nav";

export const meetingInviteCreate = gql`${Label_full}
${Organization_nav}
${Schedule_full}
${User_nav}

mutation meetingInviteCreate($input: MeetingInviteCreateInput!) {
  meetingInviteCreate(input: $input) {
    meeting {
        attendees {
            id
            isBot
            name
            handle
        }
        labels {
            ...Label_full
        }
        schedule {
            ...Schedule_full
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

