import gql from 'graphql-tag';
import { Label_full } from '../fragments/Label_full';
import { Label_list } from '../fragments/Label_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { Tag_list } from '../fragments/Tag_list';
import { User_nav } from '../fragments/User_nav';

export const routineVersionFindOne = gql`${Label_full}
${Label_list}
${Organization_nav}
${Tag_list}
${User_nav}

query routineVersion($input: FindVersionInput!) {
  routineVersion(input: $input) {
    versionNotes
    apiVersion {
        pullRequest {
            id
            created_at
            updated_at
            mergedOrRejectedAt
            commentsCount
            status
            createdBy {
                id
                name
                handle
            }
            you {
                canComment
                canDelete
                canReport
                canUpdate
            }
        }
        root {
            parent {
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
                    details
                    name
                    summary
                }
            }
            stats {
                id
                periodStart
                periodEnd
                periodType
                calls
                routineVersions
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
            bookmarks
            tags {
                ...Tag_list
            }
            transfersCount
            views
            you {
                canDelete
                canBookmark
                canTransfer
                canUpdate
                canRead
                canVote
                isBookmarked
                isUpvoted
                isViewed
            }
        }
        translations {
            id
            language
            details
            name
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
            canReport
            canUpdate
            canUse
            canRead
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
                canReport
                canUpdate
                canUse
                canRead
            }
        }
        translations {
            id
            language
            description
            helpText
        }
    }
    nodes {
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
                    createdBy {
                        id
                        name
                        handle
                    }
                    you {
                        canComment
                        canDelete
                        canReport
                        canUpdate
                    }
                }
                root {
                    parent {
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
                            details
                            name
                            summary
                        }
                    }
                    stats {
                        id
                        periodStart
                        periodEnd
                        periodType
                        calls
                        routineVersions
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
                    bookmarks
                    tags {
                        ...Tag_list
                    }
                    transfersCount
                    views
                    you {
                        canDelete
                        canBookmark
                        canTransfer
                        canUpdate
                        canRead
                        canVote
                        isBookmarked
                        isUpvoted
                        isViewed
                    }
                }
                translations {
                    id
                    language
                    details
                    name
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
                    canReport
                    canUpdate
                    canUse
                    canRead
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
                        canReport
                        canUpdate
                        canUse
                        canRead
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
                        canReport
                        canUpdate
                        canUse
                        canRead
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
                createdBy {
                    id
                    name
                    handle
                }
                you {
                    canComment
                    canDelete
                    canReport
                    canUpdate
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
                    canVote
                    isBookmarked
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
                    createdBy {
                        id
                        name
                        handle
                    }
                    you {
                        canComment
                        canDelete
                        canReport
                        canUpdate
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
                    parent {
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
                            jsonVariable
                            name
                        }
                    }
                    stats {
                        id
                        periodStart
                        periodEnd
                        periodType
                        calls
                        routineVersions
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
                    bookmarks
                    tags {
                        ...Tag_list
                    }
                    transfersCount
                    views
                    you {
                        canDelete
                        canBookmark
                        canTransfer
                        canUpdate
                        canRead
                        canVote
                        isBookmarked
                        isUpvoted
                        isViewed
                    }
                }
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
                    canReport
                    canUpdate
                    canUse
                    canRead
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
    nodeLinks {
        id
        operation
        whens {
            id
            condition
            translations {
                id
                language
                description
                name
            }
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
                canReport
                canUpdate
                canUse
                canRead
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
        createdBy {
            id
            name
            handle
        }
        you {
            canComment
            canDelete
            canReport
            canUpdate
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
            canVote
            isBookmarked
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
            createdBy {
                id
                name
                handle
            }
            you {
                canComment
                canDelete
                canReport
                canUpdate
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
            parent {
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
                    jsonVariable
                    name
                }
            }
            stats {
                id
                periodStart
                periodEnd
                periodType
                calls
                routineVersions
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
            bookmarks
            tags {
                ...Tag_list
            }
            transfersCount
            views
            you {
                canDelete
                canBookmark
                canTransfer
                canUpdate
                canRead
                canVote
                isBookmarked
                isUpvoted
                isViewed
            }
        }
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
            canReport
            canUpdate
            canUse
            canRead
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
        canVote
    }
  }
}`;

export const routineVersionFindMany = gql`${Label_full}
${Organization_nav}
${User_nav}

query routineVersions($input: RoutineVersionSearchInput!) {
  routineVersions(input: $input) {
    edges {
        cursor
        node {
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
                canVote
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const routineVersionCreate = gql`${Label_full}
${Label_list}
${Organization_nav}
${Tag_list}
${User_nav}

mutation routineVersionCreate($input: RoutineVersionCreateInput!) {
  routineVersionCreate(input: $input) {
    versionNotes
    apiVersion {
        pullRequest {
            id
            created_at
            updated_at
            mergedOrRejectedAt
            commentsCount
            status
            createdBy {
                id
                name
                handle
            }
            you {
                canComment
                canDelete
                canReport
                canUpdate
            }
        }
        root {
            parent {
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
                    details
                    name
                    summary
                }
            }
            stats {
                id
                periodStart
                periodEnd
                periodType
                calls
                routineVersions
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
            bookmarks
            tags {
                ...Tag_list
            }
            transfersCount
            views
            you {
                canDelete
                canBookmark
                canTransfer
                canUpdate
                canRead
                canVote
                isBookmarked
                isUpvoted
                isViewed
            }
        }
        translations {
            id
            language
            details
            name
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
            canReport
            canUpdate
            canUse
            canRead
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
                canReport
                canUpdate
                canUse
                canRead
            }
        }
        translations {
            id
            language
            description
            helpText
        }
    }
    nodes {
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
                    createdBy {
                        id
                        name
                        handle
                    }
                    you {
                        canComment
                        canDelete
                        canReport
                        canUpdate
                    }
                }
                root {
                    parent {
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
                            details
                            name
                            summary
                        }
                    }
                    stats {
                        id
                        periodStart
                        periodEnd
                        periodType
                        calls
                        routineVersions
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
                    bookmarks
                    tags {
                        ...Tag_list
                    }
                    transfersCount
                    views
                    you {
                        canDelete
                        canBookmark
                        canTransfer
                        canUpdate
                        canRead
                        canVote
                        isBookmarked
                        isUpvoted
                        isViewed
                    }
                }
                translations {
                    id
                    language
                    details
                    name
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
                    canReport
                    canUpdate
                    canUse
                    canRead
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
                        canReport
                        canUpdate
                        canUse
                        canRead
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
                        canReport
                        canUpdate
                        canUse
                        canRead
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
                createdBy {
                    id
                    name
                    handle
                }
                you {
                    canComment
                    canDelete
                    canReport
                    canUpdate
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
                    canVote
                    isBookmarked
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
                    createdBy {
                        id
                        name
                        handle
                    }
                    you {
                        canComment
                        canDelete
                        canReport
                        canUpdate
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
                    parent {
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
                            jsonVariable
                            name
                        }
                    }
                    stats {
                        id
                        periodStart
                        periodEnd
                        periodType
                        calls
                        routineVersions
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
                    bookmarks
                    tags {
                        ...Tag_list
                    }
                    transfersCount
                    views
                    you {
                        canDelete
                        canBookmark
                        canTransfer
                        canUpdate
                        canRead
                        canVote
                        isBookmarked
                        isUpvoted
                        isViewed
                    }
                }
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
                    canReport
                    canUpdate
                    canUse
                    canRead
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
    nodeLinks {
        id
        operation
        whens {
            id
            condition
            translations {
                id
                language
                description
                name
            }
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
                canReport
                canUpdate
                canUse
                canRead
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
        createdBy {
            id
            name
            handle
        }
        you {
            canComment
            canDelete
            canReport
            canUpdate
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
            canVote
            isBookmarked
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
            createdBy {
                id
                name
                handle
            }
            you {
                canComment
                canDelete
                canReport
                canUpdate
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
            parent {
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
                    jsonVariable
                    name
                }
            }
            stats {
                id
                periodStart
                periodEnd
                periodType
                calls
                routineVersions
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
            bookmarks
            tags {
                ...Tag_list
            }
            transfersCount
            views
            you {
                canDelete
                canBookmark
                canTransfer
                canUpdate
                canRead
                canVote
                isBookmarked
                isUpvoted
                isViewed
            }
        }
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
            canReport
            canUpdate
            canUse
            canRead
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
        canVote
    }
  }
}`;

export const routineVersionUpdate = gql`${Label_full}
${Label_list}
${Organization_nav}
${Tag_list}
${User_nav}

mutation routineVersionUpdate($input: RoutineVersionUpdateInput!) {
  routineVersionUpdate(input: $input) {
    versionNotes
    apiVersion {
        pullRequest {
            id
            created_at
            updated_at
            mergedOrRejectedAt
            commentsCount
            status
            createdBy {
                id
                name
                handle
            }
            you {
                canComment
                canDelete
                canReport
                canUpdate
            }
        }
        root {
            parent {
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
                    details
                    name
                    summary
                }
            }
            stats {
                id
                periodStart
                periodEnd
                periodType
                calls
                routineVersions
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
            bookmarks
            tags {
                ...Tag_list
            }
            transfersCount
            views
            you {
                canDelete
                canBookmark
                canTransfer
                canUpdate
                canRead
                canVote
                isBookmarked
                isUpvoted
                isViewed
            }
        }
        translations {
            id
            language
            details
            name
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
            canReport
            canUpdate
            canUse
            canRead
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
                canReport
                canUpdate
                canUse
                canRead
            }
        }
        translations {
            id
            language
            description
            helpText
        }
    }
    nodes {
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
                    createdBy {
                        id
                        name
                        handle
                    }
                    you {
                        canComment
                        canDelete
                        canReport
                        canUpdate
                    }
                }
                root {
                    parent {
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
                            details
                            name
                            summary
                        }
                    }
                    stats {
                        id
                        periodStart
                        periodEnd
                        periodType
                        calls
                        routineVersions
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
                    bookmarks
                    tags {
                        ...Tag_list
                    }
                    transfersCount
                    views
                    you {
                        canDelete
                        canBookmark
                        canTransfer
                        canUpdate
                        canRead
                        canVote
                        isBookmarked
                        isUpvoted
                        isViewed
                    }
                }
                translations {
                    id
                    language
                    details
                    name
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
                    canReport
                    canUpdate
                    canUse
                    canRead
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
                        canReport
                        canUpdate
                        canUse
                        canRead
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
                        canReport
                        canUpdate
                        canUse
                        canRead
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
                createdBy {
                    id
                    name
                    handle
                }
                you {
                    canComment
                    canDelete
                    canReport
                    canUpdate
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
                    canVote
                    isBookmarked
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
                    createdBy {
                        id
                        name
                        handle
                    }
                    you {
                        canComment
                        canDelete
                        canReport
                        canUpdate
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
                    parent {
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
                            jsonVariable
                            name
                        }
                    }
                    stats {
                        id
                        periodStart
                        periodEnd
                        periodType
                        calls
                        routineVersions
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
                    bookmarks
                    tags {
                        ...Tag_list
                    }
                    transfersCount
                    views
                    you {
                        canDelete
                        canBookmark
                        canTransfer
                        canUpdate
                        canRead
                        canVote
                        isBookmarked
                        isUpvoted
                        isViewed
                    }
                }
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
                    canReport
                    canUpdate
                    canUse
                    canRead
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
    nodeLinks {
        id
        operation
        whens {
            id
            condition
            translations {
                id
                language
                description
                name
            }
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
                canReport
                canUpdate
                canUse
                canRead
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
        createdBy {
            id
            name
            handle
        }
        you {
            canComment
            canDelete
            canReport
            canUpdate
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
            canVote
            isBookmarked
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
            createdBy {
                id
                name
                handle
            }
            you {
                canComment
                canDelete
                canReport
                canUpdate
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
            parent {
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
                    jsonVariable
                    name
                }
            }
            stats {
                id
                periodStart
                periodEnd
                periodType
                calls
                routineVersions
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
            bookmarks
            tags {
                ...Tag_list
            }
            transfersCount
            views
            you {
                canDelete
                canBookmark
                canTransfer
                canUpdate
                canRead
                canVote
                isBookmarked
                isUpvoted
                isViewed
            }
        }
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
            canReport
            canUpdate
            canUse
            canRead
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
        canVote
    }
  }
}`;

