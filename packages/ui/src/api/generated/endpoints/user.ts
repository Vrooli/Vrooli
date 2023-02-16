import gql from 'graphql-tag';
import { Label_full } from '../fragments/Label_full';
import { Label_list } from '../fragments/Label_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';

export const userProfile = gql`${Label_full}
${Label_list}
${Organization_nav}
${User_nav}

query profile {
  profile {
    id
    created_at
    updated_at
    handle
    isPrivate
    isPrivateApis
    isPrivateApisCreated
    isPrivateMemberships
    isPrivateOrganizationsCreated
    isPrivateProjects
    isPrivateProjectsCreated
    isPrivatePullRequests
    isPrivateQuestionsAnswered
    isPrivateQuestionsAsked
    isPrivateQuizzesCreated
    isPrivateRoles
    isPrivateRoutines
    isPrivateRoutinesCreated
    isPrivateStandards
    isPrivateStandardsCreated
    isPrivateBookmarks
    isPrivateVotes
    name
    theme
    emails {
        id
        emailAddress
        verified
    }
    pushDevices {
        id
        expires
        name
    }
    wallets {
        id
        handles {
            id
            handle
        }
        name
        publicAddress
        stakingAddress
        verified
    }
    notifications {
        id
        created_at
        category
        isRead
        title
        description
        link
        imgLink
    }
    notificationSettings
    translations {
        id
        language
        bio
    }
    schedules {
        filters {
            id
            filterType
            tag {
                id
                created_at
                tag
                bookmarks
                translations {
                    id
                    language
                    description
                }
                you {
                    isOwn
                    isBookmarked
                }
            }
            userSchedule {
                labels {
                    ...Label_list
                }
                id
                name
                description
                timeZone
                eventStart
                eventEnd
                recurring
                recurrStart
                recurrEnd
            }
        }
        labels {
            ...Label_full
        }
        reminderList {
            id
            created_at
            updated_at
            reminders {
                id
                created_at
                updated_at
                name
                description
                dueDate
                completed
                index
                reminderItems {
                    id
                    created_at
                    updated_at
                    name
                    description
                    dueDate
                    completed
                    index
                }
            }
        }
        id
        name
        description
        timeZone
        eventStart
        eventEnd
        recurring
        recurrStart
        recurrEnd
    }
    stats {
        id
        periodStart
        periodEnd
        periodType
        apisCreated
        organizationsCreated
        projectsCreated
        projectsCompleted
        projectCompletionTimeAverage
        quizzesPassed
        quizzesFailed
        routinesCreated
        routinesCompleted
        routineCompletionTimeAverage
        runProjectsStarted
        runProjectsCompleted
        runProjectCompletionTimeAverage
        runProjectContextSwitchesAverage
        runRoutinesStarted
        runRoutinesCompleted
        runRoutineCompletionTimeAverage
        runRoutineContextSwitchesAverage
        smartContractsCreated
        smartContractsCompleted
        smartContractCompletionTimeAverage
        standardsCreated
        standardsCompleted
        standardCompletionTimeAverage
    }
  }
}`;

export const userFindOne = gql`
query user($input: FindByIdOrHandleInput!) {
  user(input: $input) {
    stats {
        id
        periodStart
        periodEnd
        periodType
        apisCreated
        organizationsCreated
        projectsCreated
        projectsCompleted
        projectCompletionTimeAverage
        quizzesPassed
        quizzesFailed
        routinesCreated
        routinesCompleted
        routineCompletionTimeAverage
        runProjectsStarted
        runProjectsCompleted
        runProjectCompletionTimeAverage
        runProjectContextSwitchesAverage
        runRoutinesStarted
        runRoutinesCompleted
        runRoutineCompletionTimeAverage
        runRoutineContextSwitchesAverage
        smartContractsCreated
        smartContractsCompleted
        smartContractCompletionTimeAverage
        standardsCreated
        standardsCompleted
        standardCompletionTimeAverage
    }
    translations {
        id
        language
        bio
    }
    id
    created_at
    handle
    name
    bookmarks
    reportsReceivedCount
    you {
        canDelete
        canReport
        canUpdate
        isBookmarked
        isViewed
    }
  }
}`;

export const userFindMany = gql`
query users($input: UserSearchInput!) {
  users(input: $input) {
    edges {
        cursor
        node {
            translations {
                id
                language
                bio
            }
            id
            created_at
            handle
            name
            bookmarks
            reportsReceivedCount
            you {
                canDelete
                canReport
                canUpdate
                isBookmarked
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

export const userProfileUpdate = gql`${Label_full}
${Label_list}
${Organization_nav}
${User_nav}

mutation profileUpdate($input: ProfileUpdateInput!) {
  profileUpdate(input: $input) {
    id
    created_at
    updated_at
    handle
    isPrivate
    isPrivateApis
    isPrivateApisCreated
    isPrivateMemberships
    isPrivateOrganizationsCreated
    isPrivateProjects
    isPrivateProjectsCreated
    isPrivatePullRequests
    isPrivateQuestionsAnswered
    isPrivateQuestionsAsked
    isPrivateQuizzesCreated
    isPrivateRoles
    isPrivateRoutines
    isPrivateRoutinesCreated
    isPrivateStandards
    isPrivateStandardsCreated
    isPrivateBookmarks
    isPrivateVotes
    name
    theme
    emails {
        id
        emailAddress
        verified
    }
    pushDevices {
        id
        expires
        name
    }
    wallets {
        id
        handles {
            id
            handle
        }
        name
        publicAddress
        stakingAddress
        verified
    }
    notifications {
        id
        created_at
        category
        isRead
        title
        description
        link
        imgLink
    }
    notificationSettings
    translations {
        id
        language
        bio
    }
    schedules {
        filters {
            id
            filterType
            tag {
                id
                created_at
                tag
                bookmarks
                translations {
                    id
                    language
                    description
                }
                you {
                    isOwn
                    isBookmarked
                }
            }
            userSchedule {
                labels {
                    ...Label_list
                }
                id
                name
                description
                timeZone
                eventStart
                eventEnd
                recurring
                recurrStart
                recurrEnd
            }
        }
        labels {
            ...Label_full
        }
        reminderList {
            id
            created_at
            updated_at
            reminders {
                id
                created_at
                updated_at
                name
                description
                dueDate
                completed
                index
                reminderItems {
                    id
                    created_at
                    updated_at
                    name
                    description
                    dueDate
                    completed
                    index
                }
            }
        }
        id
        name
        description
        timeZone
        eventStart
        eventEnd
        recurring
        recurrStart
        recurrEnd
    }
    stats {
        id
        periodStart
        periodEnd
        periodType
        apisCreated
        organizationsCreated
        projectsCreated
        projectsCompleted
        projectCompletionTimeAverage
        quizzesPassed
        quizzesFailed
        routinesCreated
        routinesCompleted
        routineCompletionTimeAverage
        runProjectsStarted
        runProjectsCompleted
        runProjectCompletionTimeAverage
        runProjectContextSwitchesAverage
        runRoutinesStarted
        runRoutinesCompleted
        runRoutineCompletionTimeAverage
        runRoutineContextSwitchesAverage
        smartContractsCreated
        smartContractsCompleted
        smartContractCompletionTimeAverage
        standardsCreated
        standardsCompleted
        standardCompletionTimeAverage
    }
  }
}`;

export const userProfileEmailUpdate = gql`${Label_full}
${Label_list}
${Organization_nav}
${User_nav}

mutation profileEmailUpdate($input: ProfileEmailUpdateInput!) {
  profileEmailUpdate(input: $input) {
    id
    created_at
    updated_at
    handle
    isPrivate
    isPrivateApis
    isPrivateApisCreated
    isPrivateMemberships
    isPrivateOrganizationsCreated
    isPrivateProjects
    isPrivateProjectsCreated
    isPrivatePullRequests
    isPrivateQuestionsAnswered
    isPrivateQuestionsAsked
    isPrivateQuizzesCreated
    isPrivateRoles
    isPrivateRoutines
    isPrivateRoutinesCreated
    isPrivateStandards
    isPrivateStandardsCreated
    isPrivateBookmarks
    isPrivateVotes
    name
    theme
    emails {
        id
        emailAddress
        verified
    }
    pushDevices {
        id
        expires
        name
    }
    wallets {
        id
        handles {
            id
            handle
        }
        name
        publicAddress
        stakingAddress
        verified
    }
    notifications {
        id
        created_at
        category
        isRead
        title
        description
        link
        imgLink
    }
    notificationSettings
    translations {
        id
        language
        bio
    }
    schedules {
        filters {
            id
            filterType
            tag {
                id
                created_at
                tag
                bookmarks
                translations {
                    id
                    language
                    description
                }
                you {
                    isOwn
                    isBookmarked
                }
            }
            userSchedule {
                labels {
                    ...Label_list
                }
                id
                name
                description
                timeZone
                eventStart
                eventEnd
                recurring
                recurrStart
                recurrEnd
            }
        }
        labels {
            ...Label_full
        }
        reminderList {
            id
            created_at
            updated_at
            reminders {
                id
                created_at
                updated_at
                name
                description
                dueDate
                completed
                index
                reminderItems {
                    id
                    created_at
                    updated_at
                    name
                    description
                    dueDate
                    completed
                    index
                }
            }
        }
        id
        name
        description
        timeZone
        eventStart
        eventEnd
        recurring
        recurrStart
        recurrEnd
    }
    stats {
        id
        periodStart
        periodEnd
        periodType
        apisCreated
        organizationsCreated
        projectsCreated
        projectsCompleted
        projectCompletionTimeAverage
        quizzesPassed
        quizzesFailed
        routinesCreated
        routinesCompleted
        routineCompletionTimeAverage
        runProjectsStarted
        runProjectsCompleted
        runProjectCompletionTimeAverage
        runProjectContextSwitchesAverage
        runRoutinesStarted
        runRoutinesCompleted
        runRoutineCompletionTimeAverage
        runRoutineContextSwitchesAverage
        smartContractsCreated
        smartContractsCompleted
        smartContractCompletionTimeAverage
        standardsCreated
        standardsCompleted
        standardCompletionTimeAverage
    }
  }
}`;

export const userDeleteOne = gql`
mutation userDeleteOne($input: UserDeleteInput!) {
  userDeleteOne(input: $input) {
    success
  }
}`;

export const userExportData = gql`
mutation exportData {
  exportData
}`;

