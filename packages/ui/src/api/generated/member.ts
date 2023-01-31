import gql from 'graphql-tag';

export const findOne = gql`
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
            canEdit
            canStar
            canReport
            canView
            isStarred
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

export const findMany = gql`
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
                    canEdit
                    canStar
                    canReport
                    canView
                    isStarred
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

export const update = gql`
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
            canEdit
            canStar
            canReport
            canView
            isStarred
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

