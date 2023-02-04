import gql from 'graphql-tag';
import { Api_list } from '../fragments/Api_list';
import { Comment_list } from '../fragments/Comment_list';
import { Issue_list } from '../fragments/Issue_list';
import { Label_full } from '../fragments/Label_full';
import { Label_list } from '../fragments/Label_list';
import { Note_list } from '../fragments/Note_list';
import { Organization_list } from '../fragments/Organization_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { Post_list } from '../fragments/Post_list';
import { Project_list } from '../fragments/Project_list';
import { Question_list } from '../fragments/Question_list';
import { QuestionAnswer_list } from '../fragments/QuestionAnswer_list';
import { Quiz_list } from '../fragments/Quiz_list';
import { Routine_list } from '../fragments/Routine_list';
import { RunProject_list } from '../fragments/RunProject_list';
import { RunRoutine_list } from '../fragments/RunRoutine_list';
import { SmartContract_list } from '../fragments/SmartContract_list';
import { Standard_list } from '../fragments/Standard_list';
import { Star_list } from '../fragments/Star_list';
import { Tag_list } from '../fragments/Tag_list';
import { User_list } from '../fragments/User_list';
import { User_nav } from '../fragments/User_nav';
import { View_list } from '../fragments/View_list';

export const historyHistory = gql`${Api_list}
${Comment_list}
${Issue_list}
${Label_full}
${Label_list}
${Note_list}
${Organization_list}
${Organization_nav}
${Post_list}
${Project_list}
${Question_list}
${QuestionAnswer_list}
${Quiz_list}
${Routine_list}
${RunProject_list}
${RunRoutine_list}
${SmartContract_list}
${Standard_list}
${Star_list}
${Tag_list}
${User_list}
${User_nav}
${View_list}

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

