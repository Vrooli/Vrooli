import gql from 'graphql-tag';

export const findOne = gql`fragment Api_nav on Api {
    id
    isPrivate
}

fragment Note_nav on Note {
    id
    isPrivate
}

fragment Organization_nav on Organization {
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

fragment Project_nav on Project {
    id
    isPrivate
}

fragment Routine_nav on Routine {
    id
    isInternal
    isPrivate
}

fragment SmartContract_nav on SmartContract {
    id
    isPrivate
}

fragment Standard_nav on Standard {
    id
    isPrivate
}

fragment Label_common on Label {
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}

fragment User_nav on User {
    id
    name
    handle
}


query issue($input: FindByIdInput!) {
  issue(input: $input) {
    closedBy {
        id
        name
        handle
    }
    createdBy {
        id
        name
        handle
    }
    translations {
        id
        language
        description
        name
    }
    id
    created_at
    updated_at
    closedAt
    referencedVersionId
    status
    to {
        ... on Api {
            ...Api_nav
        }
        ... on Note {
            ...Note_nav
        }
        ... on Organization {
            ...Organization_nav
        }
        ... on Project {
            ...Project_nav
        }
        ... on Routine {
            ...Routine_nav
        }
        ... on SmartContract {
            ...SmartContract_nav
        }
        ... on Standard {
            ...Standard_nav
        }
    }
    commentsCount
    reportsCount
    score
    stars
    views
    labels {
        ...Label_common
    }
    you {
        canComment
        canDelete
        canEdit
        canStar
        canReport
        canView
        canVote
        isStarred
        isUpvoted
    }
  }
}`;

export const findMany = gql`fragment Api_nav on Api {
    id
    isPrivate
}

fragment Note_nav on Note {
    id
    isPrivate
}

fragment Organization_nav on Organization {
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

fragment Project_nav on Project {
    id
    isPrivate
}

fragment Routine_nav on Routine {
    id
    isInternal
    isPrivate
}

fragment SmartContract_nav on SmartContract {
    id
    isPrivate
}

fragment Standard_nav on Standard {
    id
    isPrivate
}

fragment Label_common on Label {
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}

fragment User_nav on User {
    id
    name
    handle
}


query issues($input: IssueSearchInput!) {
  issues(input: $input) {
    edges {
        cursor
        node {
            translations {
                id
                language
                description
                name
            }
            id
            created_at
            updated_at
            closedAt
            referencedVersionId
            status
            to {
                ... on Api {
                    ...Api_nav
                }
                ... on Note {
                    ...Note_nav
                }
                ... on Organization {
                    ...Organization_nav
                }
                ... on Project {
                    ...Project_nav
                }
                ... on Routine {
                    ...Routine_nav
                }
                ... on SmartContract {
                    ...SmartContract_nav
                }
                ... on Standard {
                    ...Standard_nav
                }
            }
            commentsCount
            reportsCount
            score
            stars
            views
            labels {
                ...Label_common
            }
            you {
                canComment
                canDelete
                canEdit
                canStar
                canReport
                canView
                canVote
                isStarred
                isUpvoted
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const create = gql`fragment Api_nav on Api {
    id
    isPrivate
}

fragment Note_nav on Note {
    id
    isPrivate
}

fragment Organization_nav on Organization {
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

fragment Project_nav on Project {
    id
    isPrivate
}

fragment Routine_nav on Routine {
    id
    isInternal
    isPrivate
}

fragment SmartContract_nav on SmartContract {
    id
    isPrivate
}

fragment Standard_nav on Standard {
    id
    isPrivate
}

fragment Label_common on Label {
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}

fragment User_nav on User {
    id
    name
    handle
}


mutation issueCreate($input: IssueCreateInput!) {
  issueCreate(input: $input) {
    closedBy {
        id
        name
        handle
    }
    createdBy {
        id
        name
        handle
    }
    translations {
        id
        language
        description
        name
    }
    id
    created_at
    updated_at
    closedAt
    referencedVersionId
    status
    to {
        ... on Api {
            ...Api_nav
        }
        ... on Note {
            ...Note_nav
        }
        ... on Organization {
            ...Organization_nav
        }
        ... on Project {
            ...Project_nav
        }
        ... on Routine {
            ...Routine_nav
        }
        ... on SmartContract {
            ...SmartContract_nav
        }
        ... on Standard {
            ...Standard_nav
        }
    }
    commentsCount
    reportsCount
    score
    stars
    views
    labels {
        ...Label_common
    }
    you {
        canComment
        canDelete
        canEdit
        canStar
        canReport
        canView
        canVote
        isStarred
        isUpvoted
    }
  }
}`;

export const update = gql`fragment Api_nav on Api {
    id
    isPrivate
}

fragment Note_nav on Note {
    id
    isPrivate
}

fragment Organization_nav on Organization {
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

fragment Project_nav on Project {
    id
    isPrivate
}

fragment Routine_nav on Routine {
    id
    isInternal
    isPrivate
}

fragment SmartContract_nav on SmartContract {
    id
    isPrivate
}

fragment Standard_nav on Standard {
    id
    isPrivate
}

fragment Label_common on Label {
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}

fragment User_nav on User {
    id
    name
    handle
}


mutation issueUpdate($input: IssueUpdateInput!) {
  issueUpdate(input: $input) {
    closedBy {
        id
        name
        handle
    }
    createdBy {
        id
        name
        handle
    }
    translations {
        id
        language
        description
        name
    }
    id
    created_at
    updated_at
    closedAt
    referencedVersionId
    status
    to {
        ... on Api {
            ...Api_nav
        }
        ... on Note {
            ...Note_nav
        }
        ... on Organization {
            ...Organization_nav
        }
        ... on Project {
            ...Project_nav
        }
        ... on Routine {
            ...Routine_nav
        }
        ... on SmartContract {
            ...SmartContract_nav
        }
        ... on Standard {
            ...Standard_nav
        }
    }
    commentsCount
    reportsCount
    score
    stars
    views
    labels {
        ...Label_common
    }
    you {
        canComment
        canDelete
        canEdit
        canStar
        canReport
        canView
        canVote
        isStarred
        isUpvoted
    }
  }
}`;

export const close = gql`fragment Api_nav on Api {
    id
    isPrivate
}

fragment Note_nav on Note {
    id
    isPrivate
}

fragment Organization_nav on Organization {
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

fragment Project_nav on Project {
    id
    isPrivate
}

fragment Routine_nav on Routine {
    id
    isInternal
    isPrivate
}

fragment SmartContract_nav on SmartContract {
    id
    isPrivate
}

fragment Standard_nav on Standard {
    id
    isPrivate
}

fragment Label_common on Label {
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}

fragment User_nav on User {
    id
    name
    handle
}


mutation issueClose($input: IssueCloseInput!) {
  issueClose(input: $input) {
    closedBy {
        id
        name
        handle
    }
    createdBy {
        id
        name
        handle
    }
    translations {
        id
        language
        description
        name
    }
    id
    created_at
    updated_at
    closedAt
    referencedVersionId
    status
    to {
        ... on Api {
            ...Api_nav
        }
        ... on Note {
            ...Note_nav
        }
        ... on Organization {
            ...Organization_nav
        }
        ... on Project {
            ...Project_nav
        }
        ... on Routine {
            ...Routine_nav
        }
        ... on SmartContract {
            ...SmartContract_nav
        }
        ... on Standard {
            ...Standard_nav
        }
    }
    commentsCount
    reportsCount
    score
    stars
    views
    labels {
        ...Label_common
    }
    you {
        canComment
        canDelete
        canEdit
        canStar
        canReport
        canView
        canVote
        isStarred
        isUpvoted
    }
  }
}`;

