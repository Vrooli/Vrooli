import gql from 'graphql-tag';

export const findOne = gql`
query role($input: FindByIdInput!) {
  role(input: $input) {
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

export const findMany = gql`
query roles($input: RoleSearchInput!) {
  roles(input: $input) {
    edges {
        cursor
        node {
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
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const create = gql`
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

export const update = gql`
mutation roleUpdate($input: RoleUpdateInput!) {
  roleUpdate(input: $input) {
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

