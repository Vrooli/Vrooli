import gql from 'graphql-tag';
import { Project_list } from '../fragments/Project_list';
import { Routine_list } from '../fragments/Routine_list';

export const projectOrRoutineFindMany = gql`${Project_list}
${Routine_list}

query projectOrRoutines($input: ProjectOrRoutineSearchInput!) {
  projectOrRoutines(input: $input) {
    edges {
        cursor
        node {
            ... on Project {
                ...Project_list
            }
            ... on Routine {
                ...Routine_list
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

