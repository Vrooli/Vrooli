import gql from 'graphql-tag';
import { Label_list } from '../fragments/Label_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';

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
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

