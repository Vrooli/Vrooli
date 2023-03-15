import gql from 'graphql-tag';
import { Api_list } from '../fragments/Api_list';
import { Api_nav } from '../fragments/Api_nav';
import { Comment_list } from '../fragments/Comment_list';
import { Issue_list } from '../fragments/Issue_list';
import { Issue_nav } from '../fragments/Issue_nav';
import { Label_common } from '../fragments/Label_common';
import { Label_full } from '../fragments/Label_full';
import { Label_list } from '../fragments/Label_list';
import { Note_list } from '../fragments/Note_list';
import { Note_nav } from '../fragments/Note_nav';
import { NoteVersion_nav } from '../fragments/NoteVersion_nav';
import { Organization_list } from '../fragments/Organization_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { Post_list } from '../fragments/Post_list';
import { Post_nav } from '../fragments/Post_nav';
import { Project_list } from '../fragments/Project_list';
import { Project_nav } from '../fragments/Project_nav';
import { ProjectVersion_nav } from '../fragments/ProjectVersion_nav';
import { PullRequest_nav } from '../fragments/PullRequest_nav';
import { Question_common } from '../fragments/Question_common';
import { Question_list } from '../fragments/Question_list';
import { QuestionAnswer_common } from '../fragments/QuestionAnswer_common';
import { QuestionAnswer_list } from '../fragments/QuestionAnswer_list';
import { Quiz_list } from '../fragments/Quiz_list';
import { Routine_list } from '../fragments/Routine_list';
import { Routine_nav } from '../fragments/Routine_nav';
import { RoutineVersion_nav } from '../fragments/RoutineVersion_nav';
import { SmartContract_list } from '../fragments/SmartContract_list';
import { SmartContract_nav } from '../fragments/SmartContract_nav';
import { SmartContractVersion_nav } from '../fragments/SmartContractVersion_nav';
import { Standard_list } from '../fragments/Standard_list';
import { Standard_nav } from '../fragments/Standard_nav';
import { StandardVersion_nav } from '../fragments/StandardVersion_nav';
import { Tag_list } from '../fragments/Tag_list';
import { User_list } from '../fragments/User_list';
import { User_nav } from '../fragments/User_nav';

export const bookmarkUpdate = gql`${Api_list}
${Api_nav}
${Comment_list}
${Issue_list}
${Issue_nav}
${Label_common}
${Label_full}
${Label_list}
${Note_list}
${Note_nav}
${NoteVersion_nav}
${Organization_list}
${Organization_nav}
${Post_list}
${Post_nav}
${Project_list}
${Project_nav}
${ProjectVersion_nav}
${PullRequest_nav}
${Question_common}
${Question_list}
${QuestionAnswer_common}
${QuestionAnswer_list}
${Quiz_list}
${Routine_list}
${Routine_nav}
${RoutineVersion_nav}
${SmartContract_list}
${SmartContract_nav}
${SmartContractVersion_nav}
${Standard_list}
${Standard_nav}
${StandardVersion_nav}
${Tag_list}
${User_list}
${User_nav}

mutation bookmarkUpdate($input: BookmarkUpdateInput!) {
  bookmarkUpdate(input: $input) {
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
        ... on Tag {
            ...Tag_list
        }
        ... on User {
            ...User_list
        }
    }
  }
}`;

