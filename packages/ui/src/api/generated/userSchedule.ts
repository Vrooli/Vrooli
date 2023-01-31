import gql from 'graphql-tag';

export const userScheduleFindOne = gql`fragment Label_full on Label {
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


query userSchedule($input: FindByIdInput!) {
  userSchedule(input: $input) {
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
}`;

export const userScheduleFindMany = gql`fragment Label_list on Label {
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


query userSchedules($input: UserScheduleSearchInput!) {
  userSchedules(input: $input) {
    edges {
        cursor
        node {
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
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const userScheduleCreate = gql`fragment Label_full on Label {
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


mutation userScheduleCreate($input: UserScheduleCreateInput!) {
  userScheduleCreate(input: $input) {
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
}`;

export const userScheduleUpdate = gql`fragment Label_full on Label {
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


mutation userScheduleUpdate($input: UserScheduleUpdateInput!) {
  userScheduleUpdate(input: $input) {
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
}`;

