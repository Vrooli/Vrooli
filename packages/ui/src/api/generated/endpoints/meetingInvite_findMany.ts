import gql from "graphql-tag";
import { Label_list } from "../fragments/Label_list";
import { Organization_nav } from "../fragments/Organization_nav";
import { Schedule_list } from "../fragments/Schedule_list";
import { User_nav } from "../fragments/User_nav";

export const meetingInviteFindMany = gql`${Label_list}
${Organization_nav}
${Schedule_list}
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
                schedule {
                    ...Schedule_list
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
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

