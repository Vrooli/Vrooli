import gql from 'graphql-tag';
import { Label_list } from '../fragments/Label_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';

export const runProjectScheduleFindMany = gql`${Label_list}
${Organization_nav}
${User_nav}

query runProjectSchedules($input: RunProjectScheduleSearchInput!) {
  runProjectSchedules(input: $input) {
    edges {
        cursor
        node {
            labels {
                ...Label_list
            }
            id
            timeZone
            windowStart
            windowEnd
            recurrStart
            recurrEnd
            translations {
                id
                language
                description
                name
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

