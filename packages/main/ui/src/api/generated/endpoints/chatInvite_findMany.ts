import gql from "graphql-tag";
import { Label_list } from "../fragments/Label_list";
import { Organization_nav } from "../fragments/Organization_nav";
import { User_nav } from "../fragments/User_nav";

export const chatInviteFindMany = gql`${Label_list}
${Organization_nav}
${User_nav}

query chatInvites($input: ChatInviteSearchInput!) {
  chatInvites(input: $input) {
    edges {
        cursor
        node {
            chat {
                labels {
                    ...Label_list
                }
                translations {
                    id
                    language
                    description
                    name
                }
                id
                openToAnyoneWithInvite
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
                participantsCount
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

