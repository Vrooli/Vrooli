import gql from 'graphql-tag';

export const memberFindOne = gql`
query member($input: FindByIdInput!) {
  member(input: $input) {
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
}`;

export const memberFindMany = gql`
query members($input: MemberSearchInput!) {
  members(input: $input) {
    edges {
        cursor
        node {
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
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const memberUpdate = gql`
mutation memberUpdate($input: MemberUpdateInput!) {
  memberUpdate(input: $input) {
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
}`;

