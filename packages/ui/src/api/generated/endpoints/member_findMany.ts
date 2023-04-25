import gql from "graphql-tag";
import { Tag_list } from "../fragments/Tag_list";

export const memberFindMany = gql`${Tag_list}

query members($input: MemberSearchInput!) {
  members(input: $input) {
    edges {
        cursor
        node {
            organization {
                id
                handle
                created_at
                updated_at
                isOpenToNewMembers
                isPrivate
                commentsCount
                membersCount
                reportsCount
                bookmarks
                tags {
                    ...Tag_list
                }
                translations {
                    id
                    language
                    bio
                    name
                }
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
                translations {
                    id
                    language
                    bio
                }
                id
                created_at
                handle
                name
                bookmarks
                reportsReceivedCount
                you {
                    canDelete
                    canReport
                    canUpdate
                    isBookmarked
                    isViewed
                }
            }
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
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

