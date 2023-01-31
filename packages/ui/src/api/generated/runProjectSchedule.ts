import gql from 'graphql-tag';

export const runProjectScheduleFindOne = gql`fragment Label_full on Label {
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


query runProjectSchedule($input: FindByIdInput!) {
  runProjectSchedule(input: $input) {
    labels {
        ...Label_full
    }
    id
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
}`;

export const runProjectScheduleFindMany = gql`fragment Label_list on Label {
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


query runProjectSchedules($input: RunProjectScheduleSearchInput!) {
  runProjectSchedules(input: $input) {
    edges {
        cursor
        node {
            labels {
                ...Label_list
            }
            id
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
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const runProjectScheduleCreate = gql`fragment Label_full on Label {
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


mutation runProjectScheduleCreate($input: RunProjectScheduleCreateInput!) {
  runProjectScheduleCreate(input: $input) {
    labels {
        ...Label_full
    }
    id
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
}`;

export const runProjectScheduleUpdate = gql`fragment Label_full on Label {
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


mutation runProjectScheduleUpdate($input: RunProjectScheduleUpdateInput!) {
  runProjectScheduleUpdate(input: $input) {
    labels {
        ...Label_full
    }
    id
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
}`;

