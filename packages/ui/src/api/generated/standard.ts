import gql from 'graphql-tag';

export const findOne = gql`fragment Api_list on Api {
    versions {
        translations {
            id
            language
            details
            summary
        }
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

fragment Organization_nav on Organization {
    id
    handle
    you {
        canAddMembers
        canDelete
        canEdit
        canStar
        canReport
        canView
        isStarred
        isViewed
        yourMembership {
            id
            created_at
            updated_at
            isAdmin
            permissions
        }
    }
}

fragment User_nav on User {
    id
    name
    handle
}

fragment Tag_list on Tag {
    id
    created_at
    tag
    stars
    translations {
        id
        language
        description
    }
    you {
        isOwn
        isStarred
    }
}

fragment Label_list on Label {
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}

fragment ApiVersion_list on ApiVersion {
    translations {
        id
        language
        details
        summary
    }
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

fragment Note_list on Note {
    versions {
        translations {
            id
            language
            description
            text
        }
        id
        created_at
        updated_at
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

fragment NoteVersion_list on NoteVersion {
    translations {
        id
        language
        description
        text
    }
    id
    created_at
    updated_at
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

fragment Project_list on Project {
    versions {
        directories {
            translations {
                id
                language
                description
                name
            }
            id
            created_at
            updated_at
            childOrder
            isRoot
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
        }
        translations {
            id
            language
            description
            name
        }
        id
        created_at
        updated_at
        directoriesCount
        isLatest
        isPrivate
        reportsCount
        runsCount
        simplicity
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

fragment ProjectVersion_list on ProjectVersion {
    directories {
        translations {
            id
            language
            description
            name
        }
        id
        created_at
        updated_at
        childOrder
        isRoot
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
    }
    translations {
        id
        language
        description
        name
    }
    id
    created_at
    updated_at
    directoriesCount
    isLatest
    isPrivate
    reportsCount
    runsCount
    simplicity
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

fragment Routine_list on Routine {
    versions {
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

fragment Label_full on Label {
    apisCount
    issuesCount
    meetingsCount
    notesCount
    projectsCount
    routinesCount
    runProjectSchedulesCount
    runRoutineSchedulesCount
    smartContractsCount
    standardsCount
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}

fragment RoutineVersion_list on RoutineVersion {
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

fragment SmartContract_list on SmartContract {
    versions {
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

fragment SmartContractVersion_list on SmartContractVersion {
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

fragment Standard_list on Standard {
    versions {
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

fragment StandardVersion_list on StandardVersion {
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


query standard($input: FindByIdInput!) {
  standard(input: $input) {
    versions {
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
    stats {
        id
        created_at
        periodStart
        periodEnd
        periodType
        linksToInputs
        linksToOutputs
        timesUsedInCompletedRoutines
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
}`;

export const findMany = gql`fragment Organization_nav on Organization {
    id
    handle
    you {
        canAddMembers
        canDelete
        canEdit
        canStar
        canReport
        canView
        isStarred
        isViewed
        yourMembership {
            id
            created_at
            updated_at
            isAdmin
            permissions
        }
    }
}

fragment User_nav on User {
    id
    name
    handle
}

fragment Tag_list on Tag {
    id
    created_at
    tag
    stars
    translations {
        id
        language
        description
    }
    you {
        isOwn
        isStarred
    }
}

fragment Label_list on Label {
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}


query standards($input: StandardSearchInput!) {
  standards(input: $input) {
    edges {
        cursor
        node {
            versions {
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
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const create = gql`fragment Api_list on Api {
    versions {
        translations {
            id
            language
            details
            summary
        }
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

fragment Organization_nav on Organization {
    id
    handle
    you {
        canAddMembers
        canDelete
        canEdit
        canStar
        canReport
        canView
        isStarred
        isViewed
        yourMembership {
            id
            created_at
            updated_at
            isAdmin
            permissions
        }
    }
}

fragment User_nav on User {
    id
    name
    handle
}

fragment Tag_list on Tag {
    id
    created_at
    tag
    stars
    translations {
        id
        language
        description
    }
    you {
        isOwn
        isStarred
    }
}

fragment Label_list on Label {
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}

fragment ApiVersion_list on ApiVersion {
    translations {
        id
        language
        details
        summary
    }
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

fragment Note_list on Note {
    versions {
        translations {
            id
            language
            description
            text
        }
        id
        created_at
        updated_at
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

fragment NoteVersion_list on NoteVersion {
    translations {
        id
        language
        description
        text
    }
    id
    created_at
    updated_at
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

fragment Project_list on Project {
    versions {
        directories {
            translations {
                id
                language
                description
                name
            }
            id
            created_at
            updated_at
            childOrder
            isRoot
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
        }
        translations {
            id
            language
            description
            name
        }
        id
        created_at
        updated_at
        directoriesCount
        isLatest
        isPrivate
        reportsCount
        runsCount
        simplicity
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

fragment ProjectVersion_list on ProjectVersion {
    directories {
        translations {
            id
            language
            description
            name
        }
        id
        created_at
        updated_at
        childOrder
        isRoot
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
    }
    translations {
        id
        language
        description
        name
    }
    id
    created_at
    updated_at
    directoriesCount
    isLatest
    isPrivate
    reportsCount
    runsCount
    simplicity
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

fragment Routine_list on Routine {
    versions {
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

fragment Label_full on Label {
    apisCount
    issuesCount
    meetingsCount
    notesCount
    projectsCount
    routinesCount
    runProjectSchedulesCount
    runRoutineSchedulesCount
    smartContractsCount
    standardsCount
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}

fragment RoutineVersion_list on RoutineVersion {
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

fragment SmartContract_list on SmartContract {
    versions {
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

fragment SmartContractVersion_list on SmartContractVersion {
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

fragment Standard_list on Standard {
    versions {
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

fragment StandardVersion_list on StandardVersion {
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


mutation standardCreate($input: StandardCreateInput!) {
  standardCreate(input: $input) {
    versions {
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
    stats {
        id
        created_at
        periodStart
        periodEnd
        periodType
        linksToInputs
        linksToOutputs
        timesUsedInCompletedRoutines
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
}`;

export const update = gql`fragment Api_list on Api {
    versions {
        translations {
            id
            language
            details
            summary
        }
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

fragment Organization_nav on Organization {
    id
    handle
    you {
        canAddMembers
        canDelete
        canEdit
        canStar
        canReport
        canView
        isStarred
        isViewed
        yourMembership {
            id
            created_at
            updated_at
            isAdmin
            permissions
        }
    }
}

fragment User_nav on User {
    id
    name
    handle
}

fragment Tag_list on Tag {
    id
    created_at
    tag
    stars
    translations {
        id
        language
        description
    }
    you {
        isOwn
        isStarred
    }
}

fragment Label_list on Label {
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}

fragment ApiVersion_list on ApiVersion {
    translations {
        id
        language
        details
        summary
    }
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

fragment Note_list on Note {
    versions {
        translations {
            id
            language
            description
            text
        }
        id
        created_at
        updated_at
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

fragment NoteVersion_list on NoteVersion {
    translations {
        id
        language
        description
        text
    }
    id
    created_at
    updated_at
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

fragment Project_list on Project {
    versions {
        directories {
            translations {
                id
                language
                description
                name
            }
            id
            created_at
            updated_at
            childOrder
            isRoot
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
        }
        translations {
            id
            language
            description
            name
        }
        id
        created_at
        updated_at
        directoriesCount
        isLatest
        isPrivate
        reportsCount
        runsCount
        simplicity
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

fragment ProjectVersion_list on ProjectVersion {
    directories {
        translations {
            id
            language
            description
            name
        }
        id
        created_at
        updated_at
        childOrder
        isRoot
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
    }
    translations {
        id
        language
        description
        name
    }
    id
    created_at
    updated_at
    directoriesCount
    isLatest
    isPrivate
    reportsCount
    runsCount
    simplicity
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

fragment Routine_list on Routine {
    versions {
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

fragment Label_full on Label {
    apisCount
    issuesCount
    meetingsCount
    notesCount
    projectsCount
    routinesCount
    runProjectSchedulesCount
    runRoutineSchedulesCount
    smartContractsCount
    standardsCount
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}

fragment RoutineVersion_list on RoutineVersion {
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

fragment SmartContract_list on SmartContract {
    versions {
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

fragment SmartContractVersion_list on SmartContractVersion {
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

fragment Standard_list on Standard {
    versions {
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

fragment StandardVersion_list on StandardVersion {
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


mutation standardUpdate($input: StandardUpdateInput!) {
  standardUpdate(input: $input) {
    versions {
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
    stats {
        id
        created_at
        periodStart
        periodEnd
        periodType
        linksToInputs
        linksToOutputs
        timesUsedInCompletedRoutines
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
}`;

