import gql from "graphql-tag";
import { Tag_list } from "../fragments/Tag_list";

export const organizationFindMany = gql`${Tag_list}

query organizations($input: OrganizationSearchInput!) {
  organizations(input: $input) {
    edges {
        cursor
        node {
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
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

