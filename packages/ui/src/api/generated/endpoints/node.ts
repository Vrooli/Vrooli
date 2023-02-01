import gql from 'graphql-tag';
import { Api_list } from '../fragments/Api_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';
import { Tag_list } from '../fragments/Tag_list';
import { Label_list } from '../fragments/Label_list';
import { ApiVersion_list } from '../fragments/ApiVersion_list';
import { Note_list } from '../fragments/Note_list';
import { NoteVersion_list } from '../fragments/NoteVersion_list';
import { Project_list } from '../fragments/Project_list';
import { ProjectVersion_list } from '../fragments/ProjectVersion_list';
import { Routine_list } from '../fragments/Routine_list';
import { Label_full } from '../fragments/Label_full';
import { RoutineVersion_list } from '../fragments/RoutineVersion_list';
import { SmartContract_list } from '../fragments/SmartContract_list';
import { SmartContractVersion_list } from '../fragments/SmartContractVersion_list';
import { Standard_list } from '../fragments/Standard_list';
import { StandardVersion_list } from '../fragments/StandardVersion_list';

export const nodeCreate = gql`${Api_list}
${Organization_nav}
${User_nav}
${Tag_list}
${Label_list}
${ApiVersion_list}
${Note_list}
${NoteVersion_list}
${Project_list}
${ProjectVersion_list}
${Routine_list}
${Label_full}
${RoutineVersion_list}
${SmartContract_list}
${SmartContractVersion_list}
${Standard_list}
${StandardVersion_list}

mutation nodeCreate($input: NodeCreateInput!) {
  nodeCreate(input: $input) {
    id
    created_at
    updated_at
    columnIndex
    nodeType
    rowIndex
    end {
        id
        wasSuccessful
        suggestedNextRoutineVersions {
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
    routineList {
        id
        isOrdered
        isOptional
        items {
            id
            index
            isOptional
            translations {
                id
                language
                description
                name
            }
        }
    }
    routineVersion {
        versionNotes
        apiVersion {
            pullRequest {
                id
                created_at
                updated_at
                mergedOrRejectedAt
                commentsCount
                status
                from {
                    ... on ApiVersion {
                        ...ApiVersion_list
                    }
                    ... on NoteVersion {
                        ...NoteVersion_list
                    }
                    ... on ProjectVersion {
                        ...ProjectVersion_list
                    }
                    ... on RoutineVersion {
                        ...RoutineVersion_list
                    }
                    ... on SmartContractVersion {
                        ...SmartContractVersion_list
                    }
                    ... on StandardVersion {
                        ...StandardVersion_list
                    }
                }
                to {
                    ... on Api {
                        ...Api_list
                    }
                    ... on Note {
                        ...Note_list
                    }
                    ... on Project {
                        ...Project_list
                    }
                    ... on Routine {
                        ...Routine_list
                    }
                    ... on SmartContract {
                        ...SmartContract_list
                    }
                    ... on Standard {
                        ...Standard_list
                    }
                }
                createdBy {
                    id
                    name
                    handle
                }
                you {
                    canComment
                    canDelete
                    canEdit
                    canReport
                }
            }
            root {
                stats {
                    id
                    created_at
                    periodStart
                    periodEnd
                    periodType
                    calls
                }
                id
                created_at
                updated_at
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
                stars
                tags {
                    ...Tag_list
                }
                transfersCount
                views
                you {
                    canDelete
                    canEdit
                    canStar
                    canTransfer
                    canView
                    canVote
                    isStarred
                    isUpvoted
                    isViewed
                }
            }
            translations {
                id
                language
                details
                summary
            }
            versionNotes
            id
            created_at
            updated_at
            callLink
            commentsCount
            documentationLink
            forksCount
            isLatest
            isPrivate
            reportsCount
            versionIndex
            versionLabel
            you {
                canComment
                canCopy
                canDelete
                canEdit
                canReport
                canUse
                canView
            }
        }
        inputs {
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
                    canEdit
                    canReport
                    canUse
                    canView
                }
            }
            translations {
                id
                language
                description
                helpText
            }
        }
        outputs {
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
                    canEdit
                    canReport
                    canUse
                    canView
                }
            }
            translations {
                id
                language
                description
                helpText
            }
        }
        pullRequest {
            id
            created_at
            updated_at
            mergedOrRejectedAt
            commentsCount
            status
            from {
                ... on ApiVersion {
                    ...ApiVersion_list
                }
                ... on NoteVersion {
                    ...NoteVersion_list
                }
                ... on ProjectVersion {
                    ...ProjectVersion_list
                }
                ... on RoutineVersion {
                    ...RoutineVersion_list
                }
                ... on SmartContractVersion {
                    ...SmartContractVersion_list
                }
                ... on StandardVersion {
                    ...StandardVersion_list
                }
            }
            to {
                ... on Api {
                    ...Api_list
                }
                ... on Note {
                    ...Note_list
                }
                ... on Project {
                    ...Project_list
                }
                ... on Routine {
                    ...Routine_list
                }
                ... on SmartContract {
                    ...SmartContract_list
                }
                ... on Standard {
                    ...Standard_list
                }
            }
            createdBy {
                id
                name
                handle
            }
            you {
                canComment
                canDelete
                canEdit
                canReport
            }
        }
        resourceList {
            id
            created_at
            translations {
                id
                language
                description
                name
            }
            resources {
                id
                index
                link
                usedFor
                translations {
                    id
                    language
                    description
                    name
                }
            }
        }
        root {
            stats {
                id
                created_at
                periodStart
                periodEnd
                periodType
                runsStarted
                runsCompleted
                runCompletionTimeAverageInPeriod
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
            stars
            tags {
                ...Tag_list
            }
            transfersCount
            views
            you {
                canComment
                canDelete
                canEdit
                canStar
                canView
                canVote
                isStarred
                isUpvoted
                isViewed
            }
        }
        smartContractVersion {
            versionNotes
            pullRequest {
                id
                created_at
                updated_at
                mergedOrRejectedAt
                commentsCount
                status
                from {
                    ... on ApiVersion {
                        ...ApiVersion_list
                    }
                    ... on NoteVersion {
                        ...NoteVersion_list
                    }
                    ... on ProjectVersion {
                        ...ProjectVersion_list
                    }
                    ... on RoutineVersion {
                        ...RoutineVersion_list
                    }
                    ... on SmartContractVersion {
                        ...SmartContractVersion_list
                    }
                    ... on StandardVersion {
                        ...StandardVersion_list
                    }
                }
                to {
                    ... on Api {
                        ...Api_list
                    }
                    ... on Note {
                        ...Note_list
                    }
                    ... on Project {
                        ...Project_list
                    }
                    ... on Routine {
                        ...Routine_list
                    }
                    ... on SmartContract {
                        ...SmartContract_list
                    }
                    ... on Standard {
                        ...Standard_list
                    }
                }
                createdBy {
                    id
                    name
                    handle
                }
                you {
                    canComment
                    canDelete
                    canEdit
                    canReport
                }
            }
            resourceList {
                id
                created_at
                translations {
                    id
                    language
                    description
                    name
                }
                resources {
                    id
                    index
                    link
                    usedFor
                    translations {
                        id
                        language
                        description
                        name
                    }
                }
            }
            root {
                stats {
                    id
                    created_at
                    periodStart
                    periodEnd
                    periodType
                    calls
                }
                id
                created_at
                updated_at
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
                stars
                tags {
                    ...Tag_list
                }
                transfersCount
                views
                you {
                    canDelete
                    canEdit
                    canStar
                    canTransfer
                    canView
                    canVote
                    isStarred
                    isUpvoted
                    isViewed
                }
            }
            translations {
                id
                language
                description
                jsonVariable
            }
            id
            created_at
            updated_at
            isComplete
            isDeleted
            isLatest
            isPrivate
            default
            contractType
            content
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
                canEdit
                canReport
                canUse
                canView
            }
        }
        suggestedNextByRoutineVersion {
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
                                canEdit
                                canReport
                                canUse
                                canView
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
                runRoutineSchedule {
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
                user {
                    ...User_nav
                }
                you {
                    canDelete
                    canEdit
                    canView
                }
            }
            canComment
            canCopy
            canDelete
            canEdit
            canStar
            canReport
            canRun
            canView
            canVote
        }
    }
    translations {
        id
        language
        description
        name
    }
  }
}`;

export const nodeUpdate = gql`${Api_list}
${Organization_nav}
${User_nav}
${Tag_list}
${Label_list}
${ApiVersion_list}
${Note_list}
${NoteVersion_list}
${Project_list}
${ProjectVersion_list}
${Routine_list}
${Label_full}
${RoutineVersion_list}
${SmartContract_list}
${SmartContractVersion_list}
${Standard_list}
${StandardVersion_list}

mutation nodeUpdate($input: NodeUpdateInput!) {
  nodeUpdate(input: $input) {
    id
    created_at
    updated_at
    columnIndex
    nodeType
    rowIndex
    end {
        id
        wasSuccessful
        suggestedNextRoutineVersions {
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
    routineList {
        id
        isOrdered
        isOptional
        items {
            id
            index
            isOptional
            translations {
                id
                language
                description
                name
            }
        }
    }
    routineVersion {
        versionNotes
        apiVersion {
            pullRequest {
                id
                created_at
                updated_at
                mergedOrRejectedAt
                commentsCount
                status
                from {
                    ... on ApiVersion {
                        ...ApiVersion_list
                    }
                    ... on NoteVersion {
                        ...NoteVersion_list
                    }
                    ... on ProjectVersion {
                        ...ProjectVersion_list
                    }
                    ... on RoutineVersion {
                        ...RoutineVersion_list
                    }
                    ... on SmartContractVersion {
                        ...SmartContractVersion_list
                    }
                    ... on StandardVersion {
                        ...StandardVersion_list
                    }
                }
                to {
                    ... on Api {
                        ...Api_list
                    }
                    ... on Note {
                        ...Note_list
                    }
                    ... on Project {
                        ...Project_list
                    }
                    ... on Routine {
                        ...Routine_list
                    }
                    ... on SmartContract {
                        ...SmartContract_list
                    }
                    ... on Standard {
                        ...Standard_list
                    }
                }
                createdBy {
                    id
                    name
                    handle
                }
                you {
                    canComment
                    canDelete
                    canEdit
                    canReport
                }
            }
            root {
                stats {
                    id
                    created_at
                    periodStart
                    periodEnd
                    periodType
                    calls
                }
                id
                created_at
                updated_at
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
                stars
                tags {
                    ...Tag_list
                }
                transfersCount
                views
                you {
                    canDelete
                    canEdit
                    canStar
                    canTransfer
                    canView
                    canVote
                    isStarred
                    isUpvoted
                    isViewed
                }
            }
            translations {
                id
                language
                details
                summary
            }
            versionNotes
            id
            created_at
            updated_at
            callLink
            commentsCount
            documentationLink
            forksCount
            isLatest
            isPrivate
            reportsCount
            versionIndex
            versionLabel
            you {
                canComment
                canCopy
                canDelete
                canEdit
                canReport
                canUse
                canView
            }
        }
        inputs {
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
                    canEdit
                    canReport
                    canUse
                    canView
                }
            }
            translations {
                id
                language
                description
                helpText
            }
        }
        outputs {
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
                    canEdit
                    canReport
                    canUse
                    canView
                }
            }
            translations {
                id
                language
                description
                helpText
            }
        }
        pullRequest {
            id
            created_at
            updated_at
            mergedOrRejectedAt
            commentsCount
            status
            from {
                ... on ApiVersion {
                    ...ApiVersion_list
                }
                ... on NoteVersion {
                    ...NoteVersion_list
                }
                ... on ProjectVersion {
                    ...ProjectVersion_list
                }
                ... on RoutineVersion {
                    ...RoutineVersion_list
                }
                ... on SmartContractVersion {
                    ...SmartContractVersion_list
                }
                ... on StandardVersion {
                    ...StandardVersion_list
                }
            }
            to {
                ... on Api {
                    ...Api_list
                }
                ... on Note {
                    ...Note_list
                }
                ... on Project {
                    ...Project_list
                }
                ... on Routine {
                    ...Routine_list
                }
                ... on SmartContract {
                    ...SmartContract_list
                }
                ... on Standard {
                    ...Standard_list
                }
            }
            createdBy {
                id
                name
                handle
            }
            you {
                canComment
                canDelete
                canEdit
                canReport
            }
        }
        resourceList {
            id
            created_at
            translations {
                id
                language
                description
                name
            }
            resources {
                id
                index
                link
                usedFor
                translations {
                    id
                    language
                    description
                    name
                }
            }
        }
        root {
            stats {
                id
                created_at
                periodStart
                periodEnd
                periodType
                runsStarted
                runsCompleted
                runCompletionTimeAverageInPeriod
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
            stars
            tags {
                ...Tag_list
            }
            transfersCount
            views
            you {
                canComment
                canDelete
                canEdit
                canStar
                canView
                canVote
                isStarred
                isUpvoted
                isViewed
            }
        }
        smartContractVersion {
            versionNotes
            pullRequest {
                id
                created_at
                updated_at
                mergedOrRejectedAt
                commentsCount
                status
                from {
                    ... on ApiVersion {
                        ...ApiVersion_list
                    }
                    ... on NoteVersion {
                        ...NoteVersion_list
                    }
                    ... on ProjectVersion {
                        ...ProjectVersion_list
                    }
                    ... on RoutineVersion {
                        ...RoutineVersion_list
                    }
                    ... on SmartContractVersion {
                        ...SmartContractVersion_list
                    }
                    ... on StandardVersion {
                        ...StandardVersion_list
                    }
                }
                to {
                    ... on Api {
                        ...Api_list
                    }
                    ... on Note {
                        ...Note_list
                    }
                    ... on Project {
                        ...Project_list
                    }
                    ... on Routine {
                        ...Routine_list
                    }
                    ... on SmartContract {
                        ...SmartContract_list
                    }
                    ... on Standard {
                        ...Standard_list
                    }
                }
                createdBy {
                    id
                    name
                    handle
                }
                you {
                    canComment
                    canDelete
                    canEdit
                    canReport
                }
            }
            resourceList {
                id
                created_at
                translations {
                    id
                    language
                    description
                    name
                }
                resources {
                    id
                    index
                    link
                    usedFor
                    translations {
                        id
                        language
                        description
                        name
                    }
                }
            }
            root {
                stats {
                    id
                    created_at
                    periodStart
                    periodEnd
                    periodType
                    calls
                }
                id
                created_at
                updated_at
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
                stars
                tags {
                    ...Tag_list
                }
                transfersCount
                views
                you {
                    canDelete
                    canEdit
                    canStar
                    canTransfer
                    canView
                    canVote
                    isStarred
                    isUpvoted
                    isViewed
                }
            }
            translations {
                id
                language
                description
                jsonVariable
            }
            id
            created_at
            updated_at
            isComplete
            isDeleted
            isLatest
            isPrivate
            default
            contractType
            content
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
                canEdit
                canReport
                canUse
                canView
            }
        }
        suggestedNextByRoutineVersion {
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
                                canEdit
                                canReport
                                canUse
                                canView
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
                runRoutineSchedule {
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
                user {
                    ...User_nav
                }
                you {
                    canDelete
                    canEdit
                    canView
                }
            }
            canComment
            canCopy
            canDelete
            canEdit
            canStar
            canReport
            canRun
            canView
            canVote
        }
    }
    translations {
        id
        language
        description
        name
    }
  }
}`;

