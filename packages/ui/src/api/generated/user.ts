import gql from 'graphql-tag';

export const profile = gql`fragment Label_full on Label {
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
    isPrivateStars
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
        created_at
        periodStart
        periodEnd
        periodType
        apis
        organizations
        projects
        projectsCompleted
        projectsCompletionTimeAverageInPeriod
        quizzesPassed
        quizzesFailed
        routines
        routinesCompleted
        routinesCompletionTimeAverageInPeriod
        runsStarted
        runsCompleted
        runsCompletionTimeAverageInPeriod
        smartContractsCreated
        smartContractsCompleted
        smartContractsCompletionTimeAverageInPeriod
        standardsCreated
        standardsCompleted
        standardsCompletionTimeAverageInPeriod
    }
  }
}`;

export const findOne = gql`
query user($input: FindByIdInput!) {
  user(input: $input) {
    stats {
        id
        created_at
        periodStart
        periodEnd
        periodType
        apis
        organizations
        projects
        projectsCompleted
        projectsCompletionTimeAverageInPeriod
        quizzesPassed
        quizzesFailed
        routines
        routinesCompleted
        routinesCompletionTimeAverageInPeriod
        runsStarted
        runsCompleted
        runsCompletionTimeAverageInPeriod
        smartContractsCreated
        smartContractsCompleted
        smartContractsCompletionTimeAverageInPeriod
        standardsCreated
        standardsCompleted
        standardsCompletionTimeAverageInPeriod
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
}`;

export const findMany = gql`
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
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const profileUpdate = gql`fragment Label_full on Label {
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
    isPrivateStars
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
        created_at
        periodStart
        periodEnd
        periodType
        apis
        organizations
        projects
        projectsCompleted
        projectsCompletionTimeAverageInPeriod
        quizzesPassed
        quizzesFailed
        routines
        routinesCompleted
        routinesCompletionTimeAverageInPeriod
        runsStarted
        runsCompleted
        runsCompletionTimeAverageInPeriod
        smartContractsCreated
        smartContractsCompleted
        smartContractsCompletionTimeAverageInPeriod
        standardsCreated
        standardsCompleted
        standardsCompletionTimeAverageInPeriod
    }
  }
}`;

export const profileEmailUpdate = gql`fragment Label_full on Label {
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
    isPrivateStars
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
        created_at
        periodStart
        periodEnd
        periodType
        apis
        organizations
        projects
        projectsCompleted
        projectsCompletionTimeAverageInPeriod
        quizzesPassed
        quizzesFailed
        routines
        routinesCompleted
        routinesCompletionTimeAverageInPeriod
        runsStarted
        runsCompleted
        runsCompletionTimeAverageInPeriod
        smartContractsCreated
        smartContractsCompleted
        smartContractsCompletionTimeAverageInPeriod
        standardsCreated
        standardsCompleted
        standardsCompletionTimeAverageInPeriod
    }
  }
}`;

export const deleteOne = gql`
mutation userDeleteOne($input: UserDeleteInput!) {
  userDeleteOne(input: $input) {
    success
  }
}`;

export const exportData = gql`
mutation exportData {
  exportData
}`;

