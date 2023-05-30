import gql from "graphql-tag";
import { Label_full } from "../fragments/Label_full";
import { Organization_nav } from "../fragments/Organization_nav";
import { User_nav } from "../fragments/User_nav";

export const runRoutineUpdate = gql`${Label_full}
${Organization_nav}
${User_nav}

mutation runRoutineUpdate($input: RunRoutineUpdateInput!) {
  runRoutineUpdate(input: $input) {
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
    routineVersion {
        translations {
            id
            language
            description
            instructions
            name
        }
        id
        created_at
        updated_at
        completedAt
        isAutomatable
        isComplete
        isDeleted
        isLatest
        isPrivate
        simplicity
        timesStarted
        timesCompleted
        smartContractCallData
        apiCallData
        versionIndex
        versionLabel
        commentsCount
        directoryListingsCount
        forksCount
        inputsCount
        nodesCount
        nodeLinksCount
        outputsCount
        reportsCount
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
            complexity
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
}`;

