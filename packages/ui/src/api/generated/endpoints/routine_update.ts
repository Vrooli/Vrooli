import gql from "graphql-tag";
import { ApiVersion_full } from "../fragments/ApiVersion_full";
import { Label_full } from "../fragments/Label_full";
import { Label_list } from "../fragments/Label_list";
import { Node_full } from "../fragments/Node_full";
import { NodeLink_full } from "../fragments/NodeLink_full";
import { Organization_nav } from "../fragments/Organization_nav";
import { PullRequest_full } from "../fragments/PullRequest_full";
import { ResourceList_full } from "../fragments/ResourceList_full";
import { RoutineVersion_nav } from "../fragments/RoutineVersion_nav";
import { RoutineVersionInput_full } from "../fragments/RoutineVersionInput_full";
import { RoutineVersionOutput_full } from "../fragments/RoutineVersionOutput_full";
import { SmartContractVersion_full } from "../fragments/SmartContractVersion_full";
import { Tag_list } from "../fragments/Tag_list";
import { User_nav } from "../fragments/User_nav";

export const routineUpdate = gql`${ApiVersion_full}
${Label_full}
${Label_list}
${Node_full}
${NodeLink_full}
${Organization_nav}
${PullRequest_full}
${ResourceList_full}
${RoutineVersion_nav}
${RoutineVersionInput_full}
${RoutineVersionOutput_full}
${SmartContractVersion_full}
${Tag_list}
${User_nav}

mutation routineUpdate($input: RoutineUpdateInput!) {
  routineUpdate(input: $input) {
    parent {
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
    versions {
        versionNotes
        apiVersion {
            ...ApiVersion_full
        }
        inputs {
            ...RoutineVersionInput_full
        }
        nodes {
            ...Node_full
        }
        nodeLinks {
            ...NodeLink_full
        }
        outputs {
            ...RoutineVersionOutput_full
        }
        pullRequest {
            ...PullRequest_full
        }
        resourceList {
            ...ResourceList_full
        }
        smartContractVersion {
            ...SmartContractVersion_full
        }
        suggestedNextByRoutineVersion {
            ...RoutineVersion_nav
        }
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
        complexity
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
        you {
            runs {
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
            canComment
            canCopy
            canDelete
            canBookmark
            canReport
            canRun
            canUpdate
            canRead
            canReact
        }
    }
    stats {
        id
        periodStart
        periodEnd
        periodType
        runsStarted
        runsCompleted
        runCompletionTimeAverage
        runContextSwitchesAverage
    }
    id
    created_at
    updated_at
    isInternal
    isPrivate
    issuesCount
    labels {
        ...Label_list
    }
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    permissions
    questionsCount
    score
    bookmarks
    tags {
        ...Tag_list
    }
    transfersCount
    views
    you {
        canComment
        canDelete
        canBookmark
        canUpdate
        canRead
        canReact
        isBookmarked
        isViewed
        reaction
    }
  }
}`;

