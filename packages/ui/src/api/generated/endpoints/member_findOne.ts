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

