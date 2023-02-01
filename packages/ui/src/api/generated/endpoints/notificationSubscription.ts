import gql from 'graphql-tag';
import { Api_list } from '../fragments/Api_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';
import { Tag_list } from '../fragments/Tag_list';
import { Label_list } from '../fragments/Label_list';
import { Comment_list } from '../fragments/Comment_list';
import { Api_nav } from '../fragments/Api_nav';
import { Issue_nav } from '../fragments/Issue_nav';
import { NoteVersion_nav } from '../fragments/NoteVersion_nav';
import { Post_nav } from '../fragments/Post_nav';
import { ProjectVersion_nav } from '../fragments/ProjectVersion_nav';
import { PullRequest_nav } from '../fragments/PullRequest_nav';
import { Question_common } from '../fragments/Question_common';
import { Note_nav } from '../fragments/Note_nav';
import { Project_nav } from '../fragments/Project_nav';
import { Routine_nav } from '../fragments/Routine_nav';
import { SmartContract_nav } from '../fragments/SmartContract_nav';
import { Standard_nav } from '../fragments/Standard_nav';
import { QuestionAnswer_common } from '../fragments/QuestionAnswer_common';
import { RoutineVersion_nav } from '../fragments/RoutineVersion_nav';
import { SmartContractVersion_nav } from '../fragments/SmartContractVersion_nav';
import { StandardVersion_nav } from '../fragments/StandardVersion_nav';
import { Issue_list } from '../fragments/Issue_list';
import { Label_common } from '../fragments/Label_common';
import { Meeting_list } from '../fragments/Meeting_list';
import { Note_list } from '../fragments/Note_list';
import { Organization_list } from '../fragments/Organization_list';
import { Project_list } from '../fragments/Project_list';
import { PullRequest_list } from '../fragments/PullRequest_list';
import { ApiVersion_list } from '../fragments/ApiVersion_list';
import { NoteVersion_list } from '../fragments/NoteVersion_list';
import { ProjectVersion_list } from '../fragments/ProjectVersion_list';
import { Routine_list } from '../fragments/Routine_list';
import { Label_full } from '../fragments/Label_full';
import { RoutineVersion_list } from '../fragments/RoutineVersion_list';
import { SmartContract_list } from '../fragments/SmartContract_list';
import { SmartContractVersion_list } from '../fragments/SmartContractVersion_list';
import { Standard_list } from '../fragments/Standard_list';
import { StandardVersion_list } from '../fragments/StandardVersion_list';
import { Question_list } from '../fragments/Question_list';
import { Quiz_list } from '../fragments/Quiz_list';
import { Report_list } from '../fragments/Report_list';

export const notificationSubscriptionFindOne = gql`${Api_list}
${Organization_nav}
${User_nav}
${Tag_list}
${Label_list}
${Comment_list}
${Api_nav}
${Issue_nav}
${NoteVersion_nav}
${Post_nav}
${ProjectVersion_nav}
${PullRequest_nav}
${Question_common}
${Note_nav}
${Project_nav}
${Routine_nav}
${SmartContract_nav}
${Standard_nav}
${QuestionAnswer_common}
${RoutineVersion_nav}
${SmartContractVersion_nav}
${StandardVersion_nav}
${Issue_list}
${Label_common}
${Meeting_list}
${Note_list}
${Organization_list}
${Project_list}
${PullRequest_list}
${ApiVersion_list}
${NoteVersion_list}
${ProjectVersion_list}
${Routine_list}
${Label_full}
${RoutineVersion_list}
${SmartContract_list}
${SmartContractVersion_list}
${Standard_list}
${StandardVersion_list}
${Question_list}
${Quiz_list}
${Report_list}

query notificationSubscription($input: FindByIdInput!) {
  notificationSubscription(input: $input) {
    id
    created_at
    silent
    object {
        ... on Api {
            ...Api_list
        }
        ... on Comment {
            ...Comment_list
        }
        ... on Issue {
            ...Issue_list
        }
        ... on Meeting {
            ...Meeting_list
        }
        ... on Note {
            ...Note_list
        }
        ... on Organization {
            ...Organization_list
        }
        ... on Project {
            ...Project_list
        }
        ... on PullRequest {
            ...PullRequest_list
        }
        ... on Question {
            ...Question_list
        }
        ... on Quiz {
            ...Quiz_list
        }
        ... on Report {
            ...Report_list
        }
        ... on Routine {
            ...Routine_list
        }
        ... on SmartContract {
            ...SmartContract_list
        }
        ... on Standard {
            ...Standard_list
        }
    }
  }
}`;

export const notificationSubscriptionFindMany = gql`${Api_list}
${Organization_nav}
${User_nav}
${Tag_list}
${Label_list}
${Comment_list}
${Api_nav}
${Issue_nav}
${NoteVersion_nav}
${Post_nav}
${ProjectVersion_nav}
${PullRequest_nav}
${Question_common}
${Note_nav}
${Project_nav}
${Routine_nav}
${SmartContract_nav}
${Standard_nav}
${QuestionAnswer_common}
${RoutineVersion_nav}
${SmartContractVersion_nav}
${StandardVersion_nav}
${Issue_list}
${Label_common}
${Meeting_list}
${Note_list}
${Organization_list}
${Project_list}
${PullRequest_list}
${ApiVersion_list}
${NoteVersion_list}
${ProjectVersion_list}
${Routine_list}
${Label_full}
${RoutineVersion_list}
${SmartContract_list}
${SmartContractVersion_list}
${Standard_list}
${StandardVersion_list}
${Question_list}
${Quiz_list}
${Report_list}

query notificationSubscriptions($input: NotificationSubscriptionSearchInput!) {
  notificationSubscriptions(input: $input) {
    edges {
        cursor
        node {
            id
            created_at
            silent
            object {
                ... on Api {
                    ...Api_list
                }
                ... on Comment {
                    ...Comment_list
                }
                ... on Issue {
                    ...Issue_list
                }
                ... on Meeting {
                    ...Meeting_list
                }
                ... on Note {
                    ...Note_list
                }
                ... on Organization {
                    ...Organization_list
                }
                ... on Project {
                    ...Project_list
                }
                ... on PullRequest {
                    ...PullRequest_list
                }
                ... on Question {
                    ...Question_list
                }
                ... on Quiz {
                    ...Quiz_list
                }
                ... on Report {
                    ...Report_list
                }
                ... on Routine {
                    ...Routine_list
                }
                ... on SmartContract {
                    ...SmartContract_list
                }
                ... on Standard {
                    ...Standard_list
                }
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const notificationSubscriptionCreate = gql`${Api_list}
${Organization_nav}
${User_nav}
${Tag_list}
${Label_list}
${Comment_list}
${Api_nav}
${Issue_nav}
${NoteVersion_nav}
${Post_nav}
${ProjectVersion_nav}
${PullRequest_nav}
${Question_common}
${Note_nav}
${Project_nav}
${Routine_nav}
${SmartContract_nav}
${Standard_nav}
${QuestionAnswer_common}
${RoutineVersion_nav}
${SmartContractVersion_nav}
${StandardVersion_nav}
${Issue_list}
${Label_common}
${Meeting_list}
${Note_list}
${Organization_list}
${Project_list}
${PullRequest_list}
${ApiVersion_list}
${NoteVersion_list}
${ProjectVersion_list}
${Routine_list}
${Label_full}
${RoutineVersion_list}
${SmartContract_list}
${SmartContractVersion_list}
${Standard_list}
${StandardVersion_list}
${Question_list}
${Quiz_list}
${Report_list}

mutation notificationSubscriptionCreate($input: NotificationSubscriptionCreateInput!) {
  notificationSubscriptionCreate(input: $input) {
    id
    created_at
    silent
    object {
        ... on Api {
            ...Api_list
        }
        ... on Comment {
            ...Comment_list
        }
        ... on Issue {
            ...Issue_list
        }
        ... on Meeting {
            ...Meeting_list
        }
        ... on Note {
            ...Note_list
        }
        ... on Organization {
            ...Organization_list
        }
        ... on Project {
            ...Project_list
        }
        ... on PullRequest {
            ...PullRequest_list
        }
        ... on Question {
            ...Question_list
        }
        ... on Quiz {
            ...Quiz_list
        }
        ... on Report {
            ...Report_list
        }
        ... on Routine {
            ...Routine_list
        }
        ... on SmartContract {
            ...SmartContract_list
        }
        ... on Standard {
            ...Standard_list
        }
    }
  }
}`;

export const notificationSubscriptionUpdate = gql`${Api_list}
${Organization_nav}
${User_nav}
${Tag_list}
${Label_list}
${Comment_list}
${Api_nav}
${Issue_nav}
${NoteVersion_nav}
${Post_nav}
${ProjectVersion_nav}
${PullRequest_nav}
${Question_common}
${Note_nav}
${Project_nav}
${Routine_nav}
${SmartContract_nav}
${Standard_nav}
${QuestionAnswer_common}
${RoutineVersion_nav}
${SmartContractVersion_nav}
${StandardVersion_nav}
${Issue_list}
${Label_common}
${Meeting_list}
${Note_list}
${Organization_list}
${Project_list}
${PullRequest_list}
${ApiVersion_list}
${NoteVersion_list}
${ProjectVersion_list}
${Routine_list}
${Label_full}
${RoutineVersion_list}
${SmartContract_list}
${SmartContractVersion_list}
${Standard_list}
${StandardVersion_list}
${Question_list}
${Quiz_list}
${Report_list}

mutation notificationSubscriptionUpdate($input: NotificationSubscriptionUpdateInput!) {
  notificationSubscriptionUpdate(input: $input) {
    id
    created_at
    silent
    object {
        ... on Api {
            ...Api_list
        }
        ... on Comment {
            ...Comment_list
        }
        ... on Issue {
            ...Issue_list
        }
        ... on Meeting {
            ...Meeting_list
        }
        ... on Note {
            ...Note_list
        }
        ... on Organization {
            ...Organization_list
        }
        ... on Project {
            ...Project_list
        }
        ... on PullRequest {
            ...PullRequest_list
        }
        ... on Question {
            ...Question_list
        }
        ... on Quiz {
            ...Quiz_list
        }
        ... on Report {
            ...Report_list
        }
        ... on Routine {
            ...Routine_list
        }
        ... on SmartContract {
            ...SmartContract_list
        }
        ... on Standard {
            ...Standard_list
        }
    }
  }
}`;

