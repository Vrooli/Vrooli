import gql from 'graphql-tag';
import { RunProject_list } from '../fragments/RunProject_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';
import { Label_full } from '../fragments/Label_full';
import { RunRoutine_list } from '../fragments/RunRoutine_list';
import { View_list } from '../fragments/View_list';
import { Api_list } from '../fragments/Api_list';
import { Tag_list } from '../fragments/Tag_list';
import { Label_list } from '../fragments/Label_list';
import { Issue_list } from '../fragments/Issue_list';
import { Api_nav } from '../fragments/Api_nav';
import { Note_nav } from '../fragments/Note_nav';
import { Project_nav } from '../fragments/Project_nav';
import { Routine_nav } from '../fragments/Routine_nav';
import { SmartContract_nav } from '../fragments/SmartContract_nav';
import { Standard_nav } from '../fragments/Standard_nav';
import { Label_common } from '../fragments/Label_common';
import { Note_list } from '../fragments/Note_list';
import { Organization_list } from '../fragments/Organization_list';
import { Post_list } from '../fragments/Post_list';
import { Project_list } from '../fragments/Project_list';
import { Question_list } from '../fragments/Question_list';
import { Routine_list } from '../fragments/Routine_list';
import { SmartContract_list } from '../fragments/SmartContract_list';
import { Standard_list } from '../fragments/Standard_list';
import { User_list } from '../fragments/User_list';
import { Star_list } from '../fragments/Star_list';
import { Comment_list } from '../fragments/Comment_list';
import { Issue_nav } from '../fragments/Issue_nav';
import { NoteVersion_nav } from '../fragments/NoteVersion_nav';
import { Post_nav } from '../fragments/Post_nav';
import { ProjectVersion_nav } from '../fragments/ProjectVersion_nav';
import { PullRequest_nav } from '../fragments/PullRequest_nav';
import { Question_common } from '../fragments/Question_common';
import { QuestionAnswer_common } from '../fragments/QuestionAnswer_common';
import { RoutineVersion_nav } from '../fragments/RoutineVersion_nav';
import { SmartContractVersion_nav } from '../fragments/SmartContractVersion_nav';
import { StandardVersion_nav } from '../fragments/StandardVersion_nav';
import { QuestionAnswer_list } from '../fragments/QuestionAnswer_list';
import { Quiz_list } from '../fragments/Quiz_list';

export const historyHistory = gql`${RunProject_list}
${Organization_nav}
${User_nav}
${Label_full}
${RunRoutine_list}
${View_list}
${Api_list}
${Tag_list}
${Label_list}
${Issue_list}
${Api_nav}
${Note_nav}
${Project_nav}
${Routine_nav}
${SmartContract_nav}
${Standard_nav}
${Label_common}
${Note_list}
${Organization_list}
${Post_list}
${Project_list}
${Question_list}
${Routine_list}
${SmartContract_list}
${Standard_list}
${User_list}
${Star_list}
${Comment_list}
${Issue_nav}
${NoteVersion_nav}
${Post_nav}
${ProjectVersion_nav}
${PullRequest_nav}
${Question_common}
${QuestionAnswer_common}
${RoutineVersion_nav}
${SmartContractVersion_nav}
${StandardVersion_nav}
${QuestionAnswer_list}
${Quiz_list}

query history($input: HistoryInput!) {
  history(input: $input) {
    activeRuns {
        ... on RunProject {
            ...RunProject_list
        }
        ... on RunRoutine {
            ...RunRoutine_list
        }
    }
    completedRuns {
        ... on RunProject {
            ...RunProject_list
        }
        ... on RunRoutine {
            ...RunRoutine_list
        }
    }
    recentlyViewed {
        ...View_list
    }
    recentlyStarred {
        ...Star_list
    }
  }
}`;

