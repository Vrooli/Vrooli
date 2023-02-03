import gql from 'graphql-tag';
import { Api_list } from '../fragments/Api_list';
import { Comment_list } from '../fragments/Comment_list';
import { Issue_list } from '../fragments/Issue_list';
import { Meeting_list } from '../fragments/Meeting_list';
import { Note_list } from '../fragments/Note_list';
import { Organization_list } from '../fragments/Organization_list';
import { Project_list } from '../fragments/Project_list';
import { PullRequest_list } from '../fragments/PullRequest_list';
import { Question_list } from '../fragments/Question_list';
import { Quiz_list } from '../fragments/Quiz_list';
import { Report_list } from '../fragments/Report_list';
import { Routine_list } from '../fragments/Routine_list';
import { SmartContract_list } from '../fragments/SmartContract_list';
import { Standard_list } from '../fragments/Standard_list';

export const notificationSubscriptionFindOne = gql`${Api_list}
${Comment_list}
${Issue_list}
${Meeting_list}
${Note_list}
${Organization_list}
${Project_list}
${PullRequest_list}
${Question_list}
${Quiz_list}
${Report_list}
${Routine_list}
${SmartContract_list}
${Standard_list}

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
${Comment_list}
${Issue_list}
${Meeting_list}
${Note_list}
${Organization_list}
${Project_list}
${PullRequest_list}
${Question_list}
${Quiz_list}
${Report_list}
${Routine_list}
${SmartContract_list}
${Standard_list}

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
${Comment_list}
${Issue_list}
${Meeting_list}
${Note_list}
${Organization_list}
${Project_list}
${PullRequest_list}
${Question_list}
${Quiz_list}
${Report_list}
${Routine_list}
${SmartContract_list}
${Standard_list}

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
${Comment_list}
${Issue_list}
${Meeting_list}
${Note_list}
${Organization_list}
${Project_list}
${PullRequest_list}
${Question_list}
${Quiz_list}
${Report_list}
${Routine_list}
${SmartContract_list}
${Standard_list}

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

