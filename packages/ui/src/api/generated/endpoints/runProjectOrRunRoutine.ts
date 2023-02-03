import gql from 'graphql-tag';
import { RunProject_list } from '../fragments/RunProject_list';
import { RunRoutine_list } from '../fragments/RunRoutine_list';

export const runProjectOrRunRoutineFindMany = gql`${RunProject_list}
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

