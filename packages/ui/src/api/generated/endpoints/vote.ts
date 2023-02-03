import gql from 'graphql-tag';
import { Api_list } from '../fragments/Api_list';
import { Comment_list } from '../fragments/Comment_list';
import { Issue_list } from '../fragments/Issue_list';
import { Note_list } from '../fragments/Note_list';
import { Post_list } from '../fragments/Post_list';
import { Project_list } from '../fragments/Project_list';
import { Question_list } from '../fragments/Question_list';
import { QuestionAnswer_list } from '../fragments/QuestionAnswer_list';
import { Quiz_list } from '../fragments/Quiz_list';
import { Routine_list } from '../fragments/Routine_list';
import { SmartContract_list } from '../fragments/SmartContract_list';
import { Standard_list } from '../fragments/Standard_list';

export const voteFindMany = gql`${Api_list}
${Comment_list}
${Issue_list}
${Note_list}
${Post_list}
${Project_list}
${Question_list}
${QuestionAnswer_list}
${Quiz_list}
${Routine_list}
${SmartContract_list}
${Standard_list}

query votes($input: VoteSearchInput!) {
  votes(input: $input) {
    edges {
        cursor
        node {
            id
            to {
                ... on Api {
                    ...Api_list
                }
                ... on Comment {
                    ...Comment_list
                }
                ... on Issue {
                    ...Issue_list
                }
                ... on Note {
                    ...Note_list
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
                ... on QuestionAnswer {
                    ...QuestionAnswer_list
                }
                ... on Quiz {
                    ...Quiz_list
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

export const voteVote = gql`
mutation vote($input: VoteInput!) {
  vote(input: $input) {
    success
  }
}`;

