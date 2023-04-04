import gql from 'graphql-tag';
import { Api_list } from '../fragments/Api_list';
import { Api_nav } from '../fragments/Api_nav';
import { Comment_list } from '../fragments/Comment_list';
import { Issue_list } from '../fragments/Issue_list';
import { Label_common } from '../fragments/Label_common';
import { Label_full } from '../fragments/Label_full';
import { Label_list } from '../fragments/Label_list';
import { Meeting_list } from '../fragments/Meeting_list';
import { Note_list } from '../fragments/Note_list';
import { Note_nav } from '../fragments/Note_nav';
import { Organization_list } from '../fragments/Organization_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { Project_list } from '../fragments/Project_list';
import { Project_nav } from '../fragments/Project_nav';
import { PullRequest_list } from '../fragments/PullRequest_list';
import { Question_list } from '../fragments/Question_list';
import { Quiz_list } from '../fragments/Quiz_list';
import { Report_list } from '../fragments/Report_list';
import { Routine_list } from '../fragments/Routine_list';
import { Routine_nav } from '../fragments/Routine_nav';
import { Schedule_list } from '../fragments/Schedule_list';
import { SmartContract_list } from '../fragments/SmartContract_list';
import { SmartContract_nav } from '../fragments/SmartContract_nav';
import { Standard_list } from '../fragments/Standard_list';
import { Standard_nav } from '../fragments/Standard_nav';
import { Tag_list } from '../fragments/Tag_list';
import { User_nav } from '../fragments/User_nav';

export const notificationSubscriptionFindMany = gql`${Api_list}
${Api_nav}
${Comment_list}
${Issue_list}
${Label_common}
${Label_full}
${Label_list}
${Meeting_list}
${Note_list}
${Note_nav}
${Organization_list}
${Organization_nav}
${Project_list}
${Project_nav}
${PullRequest_list}
${Question_list}
${Quiz_list}
${Report_list}
${Routine_list}
${Routine_nav}
${Schedule_list}
${SmartContract_list}
${SmartContract_nav}
${Standard_list}
${Standard_nav}
${Tag_list}
${User_nav}

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

