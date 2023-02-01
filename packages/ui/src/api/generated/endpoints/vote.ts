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
import { Note_list } from '../fragments/Note_list';
import { Post_list } from '../fragments/Post_list';
import { Project_list } from '../fragments/Project_list';
import { Question_list } from '../fragments/Question_list';
import { QuestionAnswer_list } from '../fragments/QuestionAnswer_list';
import { Quiz_list } from '../fragments/Quiz_list';
import { Routine_list } from '../fragments/Routine_list';
import { Label_full } from '../fragments/Label_full';
import { SmartContract_list } from '../fragments/SmartContract_list';
import { Standard_list } from '../fragments/Standard_list';

export const voteFindMany = gql`${Api_list}
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
${Note_list}
${Post_list}
${Project_list}
${Question_list}
${QuestionAnswer_list}
${Quiz_list}
${Routine_list}
${Label_full}
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

