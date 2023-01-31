import gql from 'graphql-tag';
import { Api_nav } from '../fragments/Api_nav';
import { Issue_nav } from '../fragments/Issue_nav';
import { NoteVersion_nav } from '../fragments/NoteVersion_nav';
import { Post_nav } from '../fragments/Post_nav';
import { ProjectVersion_nav } from '../fragments/ProjectVersion_nav';
import { PullRequest_nav } from '../fragments/PullRequest_nav';
import { Question_common } from '../fragments/Question_common';
import { Note_nav } from '../fragments/Note_nav';
import { Organization_nav } from '../fragments/Organization_nav';
import { Project_nav } from '../fragments/Project_nav';
import { Routine_nav } from '../fragments/Routine_nav';
import { SmartContract_nav } from '../fragments/SmartContract_nav';
import { Standard_nav } from '../fragments/Standard_nav';
import { QuestionAnswer_common } from '../fragments/QuestionAnswer_common';
import { RoutineVersion_nav } from '../fragments/RoutineVersion_nav';
import { SmartContractVersion_nav } from '../fragments/SmartContractVersion_nav';
import { StandardVersion_nav } from '../fragments/StandardVersion_nav';
import { User_nav } from '../fragments/User_nav';

export const questionAnswerFindOne = gql`...${Api_nav}
...${Issue_nav}
...${NoteVersion_nav}
...${Post_nav}
...${ProjectVersion_nav}
...${PullRequest_nav}
...${Question_common}
...${Note_nav}
...${Organization_nav}
...${Project_nav}
...${Routine_nav}
...${SmartContract_nav}
...${Standard_nav}
...${QuestionAnswer_common}
...${RoutineVersion_nav}
...${SmartContractVersion_nav}
...${StandardVersion_nav}
...${User_nav}

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

export const questionAnswerCreate = gql`...${Api_nav}
...${Issue_nav}
...${NoteVersion_nav}
...${Post_nav}
...${ProjectVersion_nav}
...${PullRequest_nav}
...${Question_common}
...${Note_nav}
...${Organization_nav}
...${Project_nav}
...${Routine_nav}
...${SmartContract_nav}
...${Standard_nav}
...${QuestionAnswer_common}
...${RoutineVersion_nav}
...${SmartContractVersion_nav}
...${StandardVersion_nav}
...${User_nav}

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
}`;

export const questionAnswerUpdate = gql`...${Api_nav}
...${Issue_nav}
...${NoteVersion_nav}
...${Post_nav}
...${ProjectVersion_nav}
...${PullRequest_nav}
...${Question_common}
...${Note_nav}
...${Organization_nav}
...${Project_nav}
...${Routine_nav}
...${SmartContract_nav}
...${Standard_nav}
...${QuestionAnswer_common}
...${RoutineVersion_nav}
...${SmartContractVersion_nav}
...${StandardVersion_nav}
...${User_nav}

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
}`;

export const questionAnswerAccept = gql`...${Api_nav}
...${Issue_nav}
...${NoteVersion_nav}
...${Post_nav}
...${ProjectVersion_nav}
...${PullRequest_nav}
...${Question_common}
...${Note_nav}
...${Organization_nav}
...${Project_nav}
...${Routine_nav}
...${SmartContract_nav}
...${Standard_nav}
...${QuestionAnswer_common}
...${RoutineVersion_nav}
...${SmartContractVersion_nav}
...${StandardVersion_nav}
...${User_nav}

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
}`;

