import gql from "graphql-tag";
import { Label_full } from "../fragments/Label_full";
import { Organization_nav } from "../fragments/Organization_nav";
import { User_nav } from "../fragments/User_nav";

export const runRoutineFindMany = gql`${Label_full}
${Organization_nav}
${User_nav}

query runRoutines($input: RunRoutineSearchInput!) {
  runRoutines(input: $input) {
    edges {
        cursor
        node {
            id
            isPrivate
            completedComplexity
            contextSwitches
            startedAt
            timeElapsed
            completedAt
            name
            status
            stepsCount
            inputsCount
            wasRunAutomatically
            organization {
                ...Organization_nav
            }
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
            user {
                ...User_nav
            }
            you {
                canDelete
                canUpdate
                canRead
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

