import gql from 'graphql-tag';
import { Api_nav } from '../fragments/Api_nav';
import { Note_nav } from '../fragments/Note_nav';
import { Organization_nav } from '../fragments/Organization_nav';
import { Project_nav } from '../fragments/Project_nav';
import { Routine_nav } from '../fragments/Routine_nav';
import { SmartContract_nav } from '../fragments/SmartContract_nav';
import { Standard_nav } from '../fragments/Standard_nav';
import { Tag_list } from '../fragments/Tag_list';
import { User_nav } from '../fragments/User_nav';

export const questionAnswerFindOne = gql`${Api_nav}
${Note_nav}
${Organization_nav}
${Project_nav}
${Routine_nav}
${SmartContract_nav}
${Standard_nav}
${Tag_list}
${User_nav}

query questionAnswer($input: FindByIdInput!) {
  questionAnswer(input: $input) {
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
            canStar
            canReply
            canReport
            canUpdate
            canVote
            isStarred
            isUpvoted
        }
    }
    question {
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
        reportsCount
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
        tags {
            ...Tag_list
        }
        you {
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
}`;

export const questionAnswerFindMany = gql`
query questionAnswers($input: QuestionAnswerSearchInput!) {
  questionAnswers(input: $input) {
    edges {
        cursor
        node {
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
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const questionAnswerCreate = gql`${Api_nav}
${Note_nav}
${Organization_nav}
${Project_nav}
${Routine_nav}
${SmartContract_nav}
${Standard_nav}
${Tag_list}
${User_nav}

mutation questionAnswerCreate($input: QuestionAnswerCreateInput!) {
  questionAnswerCreate(input: $input) {
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
            canStar
            canReply
            canReport
            canUpdate
            canVote
            isStarred
            isUpvoted
        }
    }
    question {
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
        reportsCount
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
        tags {
            ...Tag_list
        }
        you {
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
}`;

export const questionAnswerUpdate = gql`${Api_nav}
${Note_nav}
${Organization_nav}
${Project_nav}
${Routine_nav}
${SmartContract_nav}
${Standard_nav}
${Tag_list}
${User_nav}

mutation questionAnswerUpdate($input: QuestionAnswerUpdateInput!) {
  questionAnswerUpdate(input: $input) {
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
            canStar
            canReply
            canReport
            canUpdate
            canVote
            isStarred
            isUpvoted
        }
    }
    question {
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
        reportsCount
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
        tags {
            ...Tag_list
        }
        you {
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
}`;

export const questionAnswerAccept = gql`${Api_nav}
${Note_nav}
${Organization_nav}
${Project_nav}
${Routine_nav}
${SmartContract_nav}
${Standard_nav}
${Tag_list}
${User_nav}

mutation questionAnswerMarkAsAccepted($input: FindByIdInput!) {
  questionAnswerMarkAsAccepted(input: $input) {
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
            canStar
            canReply
            canReport
            canUpdate
            canVote
            isStarred
            isUpvoted
        }
    }
    question {
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
        reportsCount
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
        tags {
            ...Tag_list
        }
        you {
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
}`;

