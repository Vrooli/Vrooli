import gql from 'graphql-tag';
import { Label_full } from '../fragments/Label_full';
import { Label_list } from '../fragments/Label_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { Project_list } from '../fragments/Project_list';
import { Routine_list } from '../fragments/Routine_list';
import { Tag_list } from '../fragments/Tag_list';
import { User_nav } from '../fragments/User_nav';

export const projectOrRoutineFindMany = gql`${Label_full}
${Label_list}
${Organization_nav}
${Project_list}
${Routine_list}
${Tag_list}
${User_nav}

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

