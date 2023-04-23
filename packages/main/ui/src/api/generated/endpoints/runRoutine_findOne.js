import gql from "graphql-tag";
import { Label_full } from "../fragments/Label_full";
import { Organization_nav } from "../fragments/Organization_nav";
import { User_nav } from "../fragments/User_nav";
export const runRoutineFindOne = gql `${Label_full}
${Organization_nav}
${User_nav}

query runRoutine($input: FindByIdInput!) {
  runRoutine(input: $input) {
    inputs {
        id
        data
        input {
            id
            index
            isRequired
            name
            standardVersion {
                translations {
                    id
                    language
                    description
                    jsonVariable
                    name
                }
                id
                created_at
                updated_at
                isComplete
                isFile
                isLatest
                isPrivate
                default
                standardType
                props
                yup
                versionIndex
                versionLabel
                commentsCount
                directoryListingsCount
                forksCount
                reportsCount
                you {
                    canComment
                    canCopy
                    canDelete
                    canReport
                    canUpdate
                    canUse
                    canRead
                }
            }
        }
    }
    steps {
        id
        order
        contextSwitches
        startedAt
        timeElapsed
        completedAt
        name
        status
        step
        subroutine {
            id
            isAutomatable
            isComplete
            isDeleted
            isLatest
            isPrivate
            root {
                id
                isInternal
                isPrivate
            }
            translations {
                id
                language
                description
                instructions
                name
            }
            versionIndex
            versionLabel
        }
    }
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
    wasRunAutomaticaly
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
}`;
//# sourceMappingURL=runRoutine_findOne.js.map