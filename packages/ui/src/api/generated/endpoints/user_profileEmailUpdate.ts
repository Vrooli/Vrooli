import gql from 'graphql-tag';
import { Schedule_common } from '../fragments/Schedule_common';

export const userProfileEmailUpdate = gql`${Schedule_common}

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
    focusModes {
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
            focusMode {
                labels {
                    id
                    color
                    label
                }
                schedule {
                    ...Schedule_common
                }
                id
                name
                description
            }
        }
        labels {
            id
            color
            label
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
                index
                isComplete
                reminderItems {
                    id
                    created_at
                    updated_at
                    name
                    description
                    dueDate
                    index
                    isComplete
                }
            }
        }
        schedule {
            ...Schedule_common
        }
        id
        name
        description
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

