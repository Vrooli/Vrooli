import gql from 'graphql-tag';
import { Label_full } from '../fragments/Label_full';
import { Label_list } from '../fragments/Label_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { Schedule_common } from '../fragments/Schedule_common';
import { User_nav } from '../fragments/User_nav';

export const userProfileEmailUpdate = gql`${Label_full}
${Label_list}
${Organization_nav}
${Schedule_common}
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
                    ...Label_list
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

