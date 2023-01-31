import gql from 'graphql-tag';

export const questionFindOne = gql`fragment Api_nav on Api {
    id
    isPrivate
}

fragment Issue_nav on Issue {
    id
    translations {
        id
        language
        description
        name
    }
}

fragment NoteVersion_nav on NoteVersion {
    id
    isLatest
    isPrivate
    versionIndex
    versionLabel
    root {
        id
        isPrivate
    }
    translations {
        id
        language
        description
        text
    }
}

fragment Post_nav on Post {
    id
    translations {
        id
        language
        description
        name
    }
}

fragment ProjectVersion_nav on ProjectVersion {
    id
    isLatest
    isPrivate
    versionIndex
    versionLabel
    root {
        id
        isPrivate
    }
    translations {
        id
        language
        description
        name
    }
}

fragment PullRequest_nav on PullRequest {
    id
    created_at
    updated_at
    mergedOrRejectedAt
    status
}

fragment Question_common on Question {
    id
    created_at
    updated_at
    createdBy {
        id
        name
        handle
    }
    hasAcceptedAnswer
    score
    stars
    answersCount
    commentsCount
    forObject {
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
    you {
        isUpvoted
    }
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

fragment QuestionAnswer_common on QuestionAnswer {
    id
    created_at
    updated_at
    createdBy {
        id
        name
        handle
    }
    score
    stars
    isAccepted
    commentsCount
}

fragment RoutineVersion_nav on RoutineVersion {
    id
    isAutomatable
    isComplete
    isDeleted
    isLatest
    isPrivate
    root {
        id
        isInternal
        isPrivate
    }
    translations {
        id
        language
        description
        instructions
        name
    }
    versionIndex
    versionLabel
}

fragment SmartContractVersion_nav on SmartContractVersion {
    id
    isLatest
    isPrivate
    versionIndex
    versionLabel
    root {
        id
        isPrivate
    }
    translations {
        id
        language
        description
        jsonVariable
    }
}

fragment StandardVersion_nav on StandardVersion {
    id
    isLatest
    isPrivate
    versionIndex
    versionLabel
    root {
        id
        isPrivate
    }
    translations {
        id
        language
        description
        jsonVariable
    }
}

fragment User_nav on User {
    id
    name
    handle
}


query question($input: FindByIdInput!) {
  question(input: $input) {
    answers {
        comments {
            translations {
                id
                language
                text
            }
            id
            created_at
            updated_at
            owner {
                ... on Organization {
                    ...Organization_nav
                }
                ... on User {
                    ...User_nav
                }
            }
            score
            stars
            reportsCount
            you {
                canDelete
                canEdit
                canStar
                canReply
                canReport
                canVote
                isStarred
                isUpvoted
            }
        }
        translations {
            id
            language
            description
        }
        id
        created_at
        updated_at
        createdBy {
            id
            name
            handle
        }
        score
        stars
        isAccepted
        commentsCount
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
    createdBy {
        id
        name
        handle
    }
    hasAcceptedAnswer
    score
    stars
    answersCount
    commentsCount
    forObject {
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
    you {
        isUpvoted
    }
  }
}`;

export const questionFindMany = gql`fragment Api_nav on Api {
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


query questions($input: QuestionSearchInput!) {
  questions(input: $input) {
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
            createdBy {
                id
                name
                handle
            }
            hasAcceptedAnswer
            score
            stars
            answersCount
            commentsCount
            forObject {
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
            you {
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

export const questionCreate = gql`fragment Api_nav on Api {
    id
    isPrivate
}

fragment Issue_nav on Issue {
    id
    translations {
        id
        language
        description
        name
    }
}

fragment NoteVersion_nav on NoteVersion {
    id
    isLatest
    isPrivate
    versionIndex
    versionLabel
    root {
        id
        isPrivate
    }
    translations {
        id
        language
        description
        text
    }
}

fragment Post_nav on Post {
    id
    translations {
        id
        language
        description
        name
    }
}

fragment ProjectVersion_nav on ProjectVersion {
    id
    isLatest
    isPrivate
    versionIndex
    versionLabel
    root {
        id
        isPrivate
    }
    translations {
        id
        language
        description
        name
    }
}

fragment PullRequest_nav on PullRequest {
    id
    created_at
    updated_at
    mergedOrRejectedAt
    status
}

fragment Question_common on Question {
    id
    created_at
    updated_at
    createdBy {
        id
        name
        handle
    }
    hasAcceptedAnswer
    score
    stars
    answersCount
    commentsCount
    forObject {
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
    you {
        isUpvoted
    }
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

fragment QuestionAnswer_common on QuestionAnswer {
    id
    created_at
    updated_at
    createdBy {
        id
        name
        handle
    }
    score
    stars
    isAccepted
    commentsCount
}

fragment RoutineVersion_nav on RoutineVersion {
    id
    isAutomatable
    isComplete
    isDeleted
    isLatest
    isPrivate
    root {
        id
        isInternal
        isPrivate
    }
    translations {
        id
        language
        description
        instructions
        name
    }
    versionIndex
    versionLabel
}

fragment SmartContractVersion_nav on SmartContractVersion {
    id
    isLatest
    isPrivate
    versionIndex
    versionLabel
    root {
        id
        isPrivate
    }
    translations {
        id
        language
        description
        jsonVariable
    }
}

fragment StandardVersion_nav on StandardVersion {
    id
    isLatest
    isPrivate
    versionIndex
    versionLabel
    root {
        id
        isPrivate
    }
    translations {
        id
        language
        description
        jsonVariable
    }
}

fragment User_nav on User {
    id
    name
    handle
}


mutation questionCreate($input: QuestionCreateInput!) {
  questionCreate(input: $input) {
    answers {
        comments {
            translations {
                id
                language
                text
            }
            id
            created_at
            updated_at
            owner {
                ... on Organization {
                    ...Organization_nav
                }
                ... on User {
                    ...User_nav
                }
            }
            score
            stars
            reportsCount
            you {
                canDelete
                canEdit
                canStar
                canReply
                canReport
                canVote
                isStarred
                isUpvoted
            }
        }
        translations {
            id
            language
            description
        }
        id
        created_at
        updated_at
        createdBy {
            id
            name
            handle
        }
        score
        stars
        isAccepted
        commentsCount
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
    createdBy {
        id
        name
        handle
    }
    hasAcceptedAnswer
    score
    stars
    answersCount
    commentsCount
    forObject {
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
    you {
        isUpvoted
    }
  }
}`;

export const questionUpdate = gql`fragment Api_nav on Api {
    id
    isPrivate
}

fragment Issue_nav on Issue {
    id
    translations {
        id
        language
        description
        name
    }
}

fragment NoteVersion_nav on NoteVersion {
    id
    isLatest
    isPrivate
    versionIndex
    versionLabel
    root {
        id
        isPrivate
    }
    translations {
        id
        language
        description
        text
    }
}

fragment Post_nav on Post {
    id
    translations {
        id
        language
        description
        name
    }
}

fragment ProjectVersion_nav on ProjectVersion {
    id
    isLatest
    isPrivate
    versionIndex
    versionLabel
    root {
        id
        isPrivate
    }
    translations {
        id
        language
        description
        name
    }
}

fragment PullRequest_nav on PullRequest {
    id
    created_at
    updated_at
    mergedOrRejectedAt
    status
}

fragment Question_common on Question {
    id
    created_at
    updated_at
    createdBy {
        id
        name
        handle
    }
    hasAcceptedAnswer
    score
    stars
    answersCount
    commentsCount
    forObject {
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
    you {
        isUpvoted
    }
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

fragment QuestionAnswer_common on QuestionAnswer {
    id
    created_at
    updated_at
    createdBy {
        id
        name
        handle
    }
    score
    stars
    isAccepted
    commentsCount
}

fragment RoutineVersion_nav on RoutineVersion {
    id
    isAutomatable
    isComplete
    isDeleted
    isLatest
    isPrivate
    root {
        id
        isInternal
        isPrivate
    }
    translations {
        id
        language
        description
        instructions
        name
    }
    versionIndex
    versionLabel
}

fragment SmartContractVersion_nav on SmartContractVersion {
    id
    isLatest
    isPrivate
    versionIndex
    versionLabel
    root {
        id
        isPrivate
    }
    translations {
        id
        language
        description
        jsonVariable
    }
}

fragment StandardVersion_nav on StandardVersion {
    id
    isLatest
    isPrivate
    versionIndex
    versionLabel
    root {
        id
        isPrivate
    }
    translations {
        id
        language
        description
        jsonVariable
    }
}

fragment User_nav on User {
    id
    name
    handle
}


mutation questionUpdate($input: QuestionUpdateInput!) {
  questionUpdate(input: $input) {
    answers {
        comments {
            translations {
                id
                language
                text
            }
            id
            created_at
            updated_at
            owner {
                ... on Organization {
                    ...Organization_nav
                }
                ... on User {
                    ...User_nav
                }
            }
            score
            stars
            reportsCount
            you {
                canDelete
                canEdit
                canStar
                canReply
                canReport
                canVote
                isStarred
                isUpvoted
            }
        }
        translations {
            id
            language
            description
        }
        id
        created_at
        updated_at
        createdBy {
            id
            name
            handle
        }
        score
        stars
        isAccepted
        commentsCount
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
    createdBy {
        id
        name
        handle
    }
    hasAcceptedAnswer
    score
    stars
    answersCount
    commentsCount
    forObject {
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
    you {
        isUpvoted
    }
  }
}`;

