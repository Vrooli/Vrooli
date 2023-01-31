import gql from 'graphql-tag';
import { Label_full } from '../fragments/Label_full';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';

export const runProjectScheduleFindOne = gql`...${Label_full}
...${Organization_nav}
...${User_nav}

query runProjectSchedule($input: FindByIdInput!) {
  runProjectSchedule(input: $input) {
    labels {
        ...Label_full
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
}`;

export const runProjectScheduleFindMany = gql`...${Label_list}
...${Organization_nav}
...${User_nav}

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

export const runProjectScheduleCreate = gql`...${Label_full}
...${Organization_nav}
...${User_nav}

mutation runProjectScheduleCreate($input: RunProjectScheduleCreateInput!) {
  runProjectScheduleCreate(input: $input) {
    labels {
        ...Label_full
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
}`;

export const runProjectScheduleUpdate = gql`...${Label_full}
...${Organization_nav}
...${User_nav}

mutation runProjectScheduleUpdate($input: RunProjectScheduleUpdateInput!) {
  runProjectScheduleUpdate(input: $input) {
    labels {
        ...Label_full
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
}`;

