export const Routine_list = `fragment Routine_list on Routine {
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
}`;