import gql from 'graphql-tag';
import { Label_list } from '../fragments/Label_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { Schedule_common } from '../fragments/Schedule_common';
import { User_nav } from '../fragments/User_nav';

export const focusModeFindMany = gql`${Label_list}
${Organization_nav}
${Schedule_common}
${User_nav}

query focusModes($input: FocusModeSearchInput!) {
  focusModes(input: $input) {
    edges {
        cursor
        node {
            labels {
                ...Label_list
            }
            schedule {
                ...Schedule_common
            }
            id
            name
            description
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;
