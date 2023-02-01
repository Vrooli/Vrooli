import gql from 'graphql-tag';

export const memberInviteFindOne = gql`
query memberInvite($input: FindByIdInput!) {
  memberInvite(input: $input) {
    id
    created_at
    updated_at
    message
    status
    willBeAdmin
    willHavePermissions
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
    you {
        canDelete
        canEdit
    }
  }
}`;

export const memberInviteFindMany = gql`
query memberInvites($input: MemberInviteSearchInput!) {
  memberInvites(input: $input) {
    edges {
        cursor
        node {
            id
            created_at
            updated_at
            message
            status
            willBeAdmin
            willHavePermissions
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
            you {
                canDelete
                canEdit
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const memberInviteCreate = gql`
mutation memberInviteCreate($input: MemberInviteCreateInput!) {
  memberInviteCreate(input: $input) {
    id
    created_at
    updated_at
    message
    status
    willBeAdmin
    willHavePermissions
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
    you {
        canDelete
        canEdit
    }
  }
}`;

export const memberInviteUpdate = gql`
mutation memberInviteUpdate($input: MemberInviteUpdateInput!) {
  memberInviteUpdate(input: $input) {
    id
    created_at
    updated_at
    message
    status
    willBeAdmin
    willHavePermissions
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
    you {
        canDelete
        canEdit
    }
  }
}`;

export const memberInviteAccept = gql`
mutation memberInviteAccept($input: FindByIdInput!) {
  memberInviteAccept(input: $input) {
    id
    created_at
    updated_at
    message
    status
    willBeAdmin
    willHavePermissions
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
    you {
        canDelete
        canEdit
    }
  }
}`;

export const memberInviteDecline = gql`
mutation memberInviteDecline($input: FindByIdInput!) {
  memberInviteDecline(input: $input) {
    id
    created_at
    updated_at
    message
    status
    willBeAdmin
    willHavePermissions
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
    you {
        canDelete
        canEdit
    }
  }
}`;

