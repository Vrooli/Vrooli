import gql from 'graphql-tag';
import { Label_list } from '../fragments/Label_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';

export const userScheduleFindMany = gql`${Label_list}
${Organization_nav}
${User_nav}

query userSchedules($input: UserScheduleSearchInput!) {
  userSchedules(input: $input) {
    edges {
        cursor
        node {
            labels {
                ...Label_list
            }
            id
            name
            description
            timeZone
            eventStart
            eventEnd
            recurring
            recurrStart
            recurrEnd
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

