import gql from 'graphql-tag';

export const roleFindOne = gql`
query role($input: FindByIdInput!) {
  role(input: $input) {
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
    translations {
        id
        language
        description
    }
  }
}`;

