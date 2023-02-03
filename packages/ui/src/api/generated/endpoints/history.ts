import gql from 'graphql-tag';
import { Project_list } from '../fragments/Project_list';
import { Routine_list } from '../fragments/Routine_list';
import { RunProject_list } from '../fragments/RunProject_list';
import { RunRoutine_list } from '../fragments/RunRoutine_list';
import { Star_list } from '../fragments/Star_list';
import { View_list } from '../fragments/View_list';

export const historyHistory = gql`${Project_list}
${Routine_list}
${RunProject_list}
${RunRoutine_list}
${Star_list}
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

