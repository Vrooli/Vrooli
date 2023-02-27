import gql from 'graphql-tag';
import { Label_full } from '../fragments/Label_full';
import { Label_list } from '../fragments/Label_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { Tag_list } from '../fragments/Tag_list';
import { User_nav } from '../fragments/User_nav';

export const routineFindMany = gql`${Label_full}
${Label_list}
${Organization_nav}
${Tag_list}
${User_nav}

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
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

