import gql from 'graphql-tag';
import { Api_list } from '../fragments/Api_list';
import { Issue_list } from '../fragments/Issue_list';
import { Note_list } from '../fragments/Note_list';
import { Organization_list } from '../fragments/Organization_list';
import { Post_list } from '../fragments/Post_list';
import { Project_list } from '../fragments/Project_list';
import { Question_list } from '../fragments/Question_list';
import { Routine_list } from '../fragments/Routine_list';
import { SmartContract_list } from '../fragments/SmartContract_list';
import { Standard_list } from '../fragments/Standard_list';
import { User_list } from '../fragments/User_list';

export const viewFindMany = gql`${Api_list}
${Issue_list}
${Note_list}
${Organization_list}
${Post_list}
${Project_list}
${Question_list}
${Routine_list}
${SmartContract_list}
${Standard_list}
${User_list}

query views($input: ViewSearchInput!) {
  views(input: $input) {
    edges {
        cursor
        node {
            id
            to {
                ... on Api {
                    ...Api_list
                }
                ... on Issue {
                    ...Issue_list
                }
                ... on Note {
                    ...Note_list
                }
                ... on Organization {
                    ...Organization_list
                }
                ... on Post {
                    ...Post_list
                }
                ... on Project {
                    ...Project_list
                }
                ... on Question {
                    ...Question_list
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
                ... on User {
                    ...User_list
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

