import gql from 'graphql-tag';
import { RunProject_list } from '../fragments/RunProject_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';
import { Label_full } from '../fragments/Label_full';
import { RunRoutine_list } from '../fragments/RunRoutine_list';

export const runProjectOrRunRoutineFindMany = gql`${RunProject_list}
${Organization_nav}
${User_nav}
${Label_full}
${RunRoutine_list}

query runProjectOrRunRoutines($input: RunProjectOrRunRoutineSearchInput!) {
  runProjectOrRunRoutines(input: $input) {
    edges {
        cursor
        node {
            ... on RunProject {
                ...RunProject_list
            }
            ... on RunRoutine {
                ...RunRoutine_list
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

