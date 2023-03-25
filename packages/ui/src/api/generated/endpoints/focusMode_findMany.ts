import gql from 'graphql-tag';
import { Schedule_common } from '../fragments/Schedule_common';

export const focusModeFindMany = gql`${Schedule_common}

query focusModes($input: FocusModeSearchInput!) {
  focusModes(input: $input) {
    edges {
        cursor
        node {
            labels {
                id
                color
                label
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

