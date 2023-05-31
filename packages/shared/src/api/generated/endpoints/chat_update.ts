import gql from "graphql-tag";
import { Label_full } from "../fragments/Label_full";
import { Organization_nav } from "../fragments/Organization_nav";
import { User_nav } from "../fragments/User_nav";

export const chatUpdate = gql`${Label_full}
${Organization_nav}
${User_nav}

mutation chatUpdate($input: ChatUpdateInput!) {
  chatUpdate(input: $input) {
    participants {
        id
        isBot
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
    messages {
        translations {
            id
            language
            text
        }
        id
        created_at
        updated_at
        user {
            id
            isBot
            name
            handle
        }
        score
        reportsCount
        you {
            canDelete
            canReply
            canReport
            canUpdate
            canReact
            reaction
        }
    }
    labels {
        ...Label_full
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
}`;

