import gql from 'graphql-tag';
import { Label_list } from '../fragments/Label_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';

export const scheduleExceptionFindMany = gql`${Label_list}
${Organization_nav}
${User_nav}

query scheduleExceptions($input: ScheduleExceptionSearchInput!) {
  scheduleExceptions(input: $input) {
    edges {
        cursor
        node {
            schedule {
                labels {
                    ...Label_list
                }
                id
                created_at
                updated_at
                startTime
                endTime
                timezone
                recurrences {
                    id
                    recurrenceType
                    interval
                    dayOfWeek
                    dayOfMonth
                    month
                    endDate
                }
            }
            id
            originalStartTime
            newStartTime
            newEndTime
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

