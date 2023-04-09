import gql from 'graphql-tag';
import { Label_list } from '../fragments/Label_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';

export const scheduleFindMany = gql`${Label_list}
${Organization_nav}
${User_nav}

query schedules($input: ScheduleSearchInput!) {
  schedules(input: $input) {
    edges {
        cursor
        node {
            labels {
                ...Label_list
            }
            id
            created_at
            updated_at
            startTime
            endTime
            timezone
            exceptions {
                id
                originalStartTime
                newStartTime
                newEndTime
            }
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
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

