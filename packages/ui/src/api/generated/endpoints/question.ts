import gql from 'graphql-tag';
import { Api_nav } from '../fragments/Api_nav';
import { Note_nav } from '../fragments/Note_nav';
import { Organization_nav } from '../fragments/Organization_nav';
import { Project_nav } from '../fragments/Project_nav';
import { Routine_nav } from '../fragments/Routine_nav';
import { SmartContract_nav } from '../fragments/SmartContract_nav';
import { Standard_nav } from '../fragments/Standard_nav';
import { User_nav } from '../fragments/User_nav';

export const questionFindOne = gql`${Api_nav}
${Note_nav}
${Organization_nav}
${Project_nav}
${Routine_nav}
${SmartContract_nav}
${Standard_nav}
${User_nav}

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

export const questionFindMany = gql`${Api_nav}
${Note_nav}
${Organization_nav}
${Project_nav}
${Routine_nav}
${SmartContract_nav}
${Standard_nav}

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

export const questionCreate = gql`${Api_nav}
${Note_nav}
${Organization_nav}
${Project_nav}
${Routine_nav}
${SmartContract_nav}
${Standard_nav}
${User_nav}

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

export const questionUpdate = gql`${Api_nav}
${Note_nav}
${Organization_nav}
${Project_nav}
${Routine_nav}
${SmartContract_nav}
${Standard_nav}
${User_nav}

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

