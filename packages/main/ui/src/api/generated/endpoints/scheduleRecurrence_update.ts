import gql from "graphql-tag";
import { Label_full } from "../fragments/Label_full";
import { Organization_nav } from "../fragments/Organization_nav";
import { User_nav } from "../fragments/User_nav";

export const scheduleRecurrenceUpdate = gql`${Label_full}
${Organization_nav}
${User_nav}

mutation scheduleRecurrenceUpdate($input: ScheduleRecurrenceUpdateInput!) {
  scheduleRecurrenceUpdate(input: $input) {
    schedule {
        labels {
            ...Label_full
        }
        id
        created_at
        updated_at
        startTime
        endTime
        timezone
    }
    id
    recurrenceType
    interval
    dayOfWeek
    dayOfMonth
    month
    endDate
  }
}`;

