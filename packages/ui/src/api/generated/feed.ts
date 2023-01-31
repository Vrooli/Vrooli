import gql from 'graphql-tag';

export const popular = gql`fragment Organization_list on Organization {
    id
    handle
    created_at
    updated_at
    isOpenToNewMembers
    isPrivate
    commentsCount
    membersCount
    reportsCount
    stars
    tags {
        ...Tag_list
    }
    translations {
        id
        language
        bio
        name
    }
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

fragment User_list on User {
    translations {
        id
        language
        bio
    }
    id
    created_at
    handle
    name
    stars
    reportsCount
    you {
        canDelete
        canEdit
        canReport
        isStarred
        isViewed
    }
}


query popular($input: PopularInput!) {
  popular(input: $input) {
    organizations {
        ...Organization_list
    }
    projects {
        ...Project_list
    }
    routines {
        ...Routine_list
    }
    standards {
        ...Standard_list
    }
    users {
        ...User_list
    }
  }
}`;

export const learn = gql`fragment Project_list on Project {
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


query learn {
  learn {
    courses {
        ...Project_list
    }
    tutorials {
        ...Routine_list
    }
  }
}`;

export const research = gql`fragment Routine_list on Routine {
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

fragment Organization_list on Organization {
    id
    handle
    created_at
    updated_at
    isOpenToNewMembers
    isPrivate
    commentsCount
    membersCount
    reportsCount
    stars
    tags {
        ...Tag_list
    }
    translations {
        id
        language
        bio
        name
    }
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


query research {
  research {
    processes {
        ...Routine_list
    }
    newlyCompleted {
        ... on Routine {
            ...Routine_list
        }
        ... on Project {
            ...Project_list
        }
    }
    needVotes {
        ...Project_list
    }
    needInvestments {
        ...Project_list
    }
    needMembers {
        ...Organization_list
    }
  }
}`;

export const develop = gql`fragment Project_list on Project {
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


query develop {
  develop {
    completed {
        ... on Project {
            ...Project_list
        }
        ... on Routine {
            ...Routine_list
        }
    }
    inProgress {
        ... on Project {
            ...Project_list
        }
        ... on Routine {
            ...Routine_list
        }
    }
    recent {
        ... on Project {
            ...Project_list
        }
        ... on Routine {
            ...Routine_list
        }
    }
  }
}`;

