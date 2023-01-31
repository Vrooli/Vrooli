import gql from 'graphql-tag';

export const emailLogIn = gql`fragment Label_full on Label {
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


mutation emailLogIn($input: EmailLogInInput!) {
  emailLogIn(input: $input) {
    isLoggedIn
    timeZone
    users {
        handle
        hasPremium
        id
        languages
        name
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
        theme
    }
  }
}`;

export const emailSignUp = gql`fragment Label_full on Label {
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


mutation emailSignUp($input: EmailSignUpInput!) {
  emailSignUp(input: $input) {
    isLoggedIn
    timeZone
    users {
        handle
        hasPremium
        id
        languages
        name
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
        theme
    }
  }
}`;

export const emailRequestPasswordChange = gql`
mutation emailRequestPasswordChange($input: EmailRequestPasswordChangeInput!) {
  emailRequestPasswordChange(input: $input) {
    success
  }
}`;

export const emailResetPassword = gql`fragment Label_full on Label {
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


mutation emailResetPassword($input: EmailResetPasswordInput!) {
  emailResetPassword(input: $input) {
    isLoggedIn
    timeZone
    users {
        handle
        hasPremium
        id
        languages
        name
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
        theme
    }
  }
}`;

export const guestLogIn = gql`fragment Label_full on Label {
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


mutation guestLogIn {
  guestLogIn {
    isLoggedIn
    timeZone
    users {
        handle
        hasPremium
        id
        languages
        name
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
        theme
    }
  }
}`;

export const logOut = gql`fragment Label_full on Label {
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


mutation logOut($input: LogOutInput!) {
  logOut(input: $input) {
    isLoggedIn
    timeZone
    users {
        handle
        hasPremium
        id
        languages
        name
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
        theme
    }
  }
}`;

export const validateSession = gql`fragment Label_full on Label {
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


mutation validateSession($input: ValidateSessionInput!) {
  validateSession(input: $input) {
    isLoggedIn
    timeZone
    users {
        handle
        hasPremium
        id
        languages
        name
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
        theme
    }
  }
}`;

export const switchCurrentAccount = gql`fragment Label_full on Label {
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


mutation switchCurrentAccount($input: SwitchCurrentAccountInput!) {
  switchCurrentAccount(input: $input) {
    isLoggedIn
    timeZone
    users {
        handle
        hasPremium
        id
        languages
        name
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
        theme
    }
  }
}`;

export const walletInit = gql`
mutation walletInit($input: WalletInitInput!) {
  walletInit(input: $input)
}`;

export const walletComplete = gql`fragment Session_full on Session {
    isLoggedIn
    timeZone
    users {
        handle
        hasPremium
        id
        languages
        name
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
        theme
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

fragment Wallet_common on Wallet {
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


mutation walletComplete($input: WalletCompleteInput!) {
  walletComplete(input: $input) {
    firstLogIn
    session {
        ...Session_full
    }
    wallet {
        ...Wallet_common
    }
  }
}`;

