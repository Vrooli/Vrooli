import gql from 'graphql-tag';

export const routineFindOne = gql`fragment Organization_nav on Organization {
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


query routine($input: FindByIdInput!) {
  routine(input: $input) {
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
}`;

export const routineFindMany = gql`fragment Organization_nav on Organization {
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


query routines($input: RoutineSearchInput!) {
  routines(input: $input) {
    edges {
        cursor
        node {
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
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const routineCreate = gql`fragment Organization_nav on Organization {
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


mutation routineCreate($input: RoutineCreateInput!) {
  routineCreate(input: $input) {
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
}`;

export const routineUpdate = gql`fragment Organization_nav on Organization {
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


mutation routineUpdate($input: RoutineUpdateInput!) {
  routineUpdate(input: $input) {
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
}`;

