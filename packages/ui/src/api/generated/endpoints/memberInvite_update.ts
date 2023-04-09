import gql from 'graphql-tag';

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
    you {
        canDelete
        canUpdate
    }
  }
}`;

