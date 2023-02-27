import gql from 'graphql-tag';

export const roleCreate = gql`
mutation roleCreate($input: RoleCreateInput!) {
  roleCreate(input: $input) {
    members {
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
    id
    created_at
    updated_at
    name
    permissions
    membersCount
    translations {
        id
        language
        description
    }
  }
}`;

