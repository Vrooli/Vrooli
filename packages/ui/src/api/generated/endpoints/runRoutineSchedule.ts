import gql from 'graphql-tag';
import { Label_full } from '../fragments/Label_full';
import { Label_list } from '../fragments/Label_list';

export const runRoutineScheduleFindOne = gql`${Label_full}

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

