import gql from 'graphql-tag';
import { Label_full } from '../fragments/Label_full';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';

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

