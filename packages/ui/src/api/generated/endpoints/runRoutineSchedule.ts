import gql from 'graphql-tag';
import { Label_full } from '../fragments/Label_full';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';
import { Label_list } from '../fragments/Label_list';

export const runRoutineScheduleFindOne = gql`${Label_full}
${Organization_nav}
${User_nav}

query runRoutineSchedule($input: FindByIdInput!) {
  runRoutineSchedule(input: $input) {
    labels {
        ...Label_full
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
}`;

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

export const runRoutineScheduleCreate = gql`${Label_full}
${Organization_nav}
${User_nav}

mutation runRoutineScheduleCreate($input: RunRoutineScheduleCreateInput!) {
  runRoutineScheduleCreate(input: $input) {
    labels {
        ...Label_full
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
}`;

export const runRoutineScheduleUpdate = gql`${Label_full}
${Organization_nav}
${User_nav}

mutation runRoutineScheduleUpdate($input: RunRoutineScheduleUpdateInput!) {
  runRoutineScheduleUpdate(input: $input) {
    labels {
        ...Label_full
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
}`;

