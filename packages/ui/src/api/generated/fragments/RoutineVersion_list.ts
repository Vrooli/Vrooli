export const RoutineVersion_list = `fragment RoutineVersion_list on RoutineVersion {
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
    canStar
    canReport
    canRun
    canUpdate
    canRead
    canVote
}
}`;