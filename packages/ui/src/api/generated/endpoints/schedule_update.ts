import gql from 'graphql-tag';
import { Label_full } from '../fragments/Label_full';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';

export const scheduleUpdate = gql`${Label_full}
${Organization_nav}
${User_nav}

mutation scheduleUpdate($input: ScheduleUpdateInput!) {
  scheduleUpdate(input: $input) {
    labels {
        ...Label_full
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
}`;

