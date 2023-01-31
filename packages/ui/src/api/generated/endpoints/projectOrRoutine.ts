import gql from 'graphql-tag';
import { Project_list } from '../fragments/Project_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';
import { Tag_list } from '../fragments/Tag_list';
import { Label_list } from '../fragments/Label_list';
import { Routine_list } from '../fragments/Routine_list';
import { Label_full } from '../fragments/Label_full';

export const projectOrRoutineFindMany = gql`...${Project_list}
...${Organization_nav}
...${User_nav}
...${Tag_list}
...${Label_list}
...${Routine_list}
...${Label_full}

query projectOrRoutines($input: ProjectOrRoutineSearchInput!) {
  projectOrRoutines(input: $input) {
    edges {
        cursor
        node {
            ... on Project {
            }
            ... on Routine {
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

