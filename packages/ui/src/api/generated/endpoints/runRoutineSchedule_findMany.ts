import gql from 'graphql-tag';
import { Label_list } from '../fragments/Label_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';

export const runRoutineScheduleFindMany = gql`${Label_list}
${Organization_nav}
${User_nav}

query runRoutineSchedules($input: RunRoutineScheduleSearchInput!) {
  runRoutineSchedules(input: $input) {
    edges {
        cursor
        node {
            labels {
                ...Label_list
            }
            id
            attemptAutomatic
            maxAutomaticAttempts
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

