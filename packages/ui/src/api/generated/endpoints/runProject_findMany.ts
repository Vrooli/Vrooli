import gql from 'graphql-tag';
import { Label_full } from '../fragments/Label_full';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';

export const runProjectFindMany = gql`${Label_full}
${Organization_nav}
${User_nav}

query runProjects($input: RunProjectSearchInput!) {
  runProjects(input: $input) {
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
            organization {
                ...Organization_nav
            }
            projectVersion {
                id
                isLatest
                isPrivate
                versionIndex
                versionLabel
                root {
                    id
                    isPrivate
                }
                translations {
                    id
                    language
                    description
                    name
                }
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

