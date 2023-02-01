import gql from 'graphql-tag';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';
import { Label_full } from '../fragments/Label_full';

export const runProjectFindOne = gql`${Organization_nav}
${User_nav}
${Label_full}

query runProject($input: FindByIdInput!) {
  runProject(input: $input) {
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
        directory {
            id
            created_at
            updated_at
            childOrder
            isRoot
            projectVersion {
                id
                isLatest
                isPrivate
                versionIndex
                versionLabel
                root {
                    id
                    isPrivate
                }
                translations {
                    id
                    language
                    description
                    name
                }
            }
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
    wasRunAutomaticaly
    organization {
        ...Organization_nav
    }
    projectVersion {
        id
        isLatest
        isPrivate
        versionIndex
        versionLabel
        root {
            id
            isPrivate
        }
        translations {
            id
            language
            description
            name
        }
    }
    runProjectSchedule {
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
    user {
        ...User_nav
    }
    you {
        canDelete
        canEdit
        canView
    }
  }
}`;

export const runProjectFindMany = gql`${Organization_nav}
${User_nav}
${Label_full}

query runProjects($input: RunProjectSearchInput!) {
  runProjects(input: $input) {
    edges {
        cursor
        node {
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
            wasRunAutomaticaly
            organization {
                ...Organization_nav
            }
            projectVersion {
                id
                isLatest
                isPrivate
                versionIndex
                versionLabel
                root {
                    id
                    isPrivate
                }
                translations {
                    id
                    language
                    description
                    name
                }
            }
            runProjectSchedule {
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
            user {
                ...User_nav
            }
            you {
                canDelete
                canEdit
                canView
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const runProjectCreate = gql`${Organization_nav}
${User_nav}
${Label_full}

mutation runProjectCreate($input: RunProjectCreateInput!) {
  runProjectCreate(input: $input) {
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
        directory {
            id
            created_at
            updated_at
            childOrder
            isRoot
            projectVersion {
                id
                isLatest
                isPrivate
                versionIndex
                versionLabel
                root {
                    id
                    isPrivate
                }
                translations {
                    id
                    language
                    description
                    name
                }
            }
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
    wasRunAutomaticaly
    organization {
        ...Organization_nav
    }
    projectVersion {
        id
        isLatest
        isPrivate
        versionIndex
        versionLabel
        root {
            id
            isPrivate
        }
        translations {
            id
            language
            description
            name
        }
    }
    runProjectSchedule {
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
    user {
        ...User_nav
    }
    you {
        canDelete
        canEdit
        canView
    }
  }
}`;

export const runProjectUpdate = gql`${Organization_nav}
${User_nav}
${Label_full}

mutation runProjectUpdate($input: RunProjectUpdateInput!) {
  runProjectUpdate(input: $input) {
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
        directory {
            id
            created_at
            updated_at
            childOrder
            isRoot
            projectVersion {
                id
                isLatest
                isPrivate
                versionIndex
                versionLabel
                root {
                    id
                    isPrivate
                }
                translations {
                    id
                    language
                    description
                    name
                }
            }
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
    wasRunAutomaticaly
    organization {
        ...Organization_nav
    }
    projectVersion {
        id
        isLatest
        isPrivate
        versionIndex
        versionLabel
        root {
            id
            isPrivate
        }
        translations {
            id
            language
            description
            name
        }
    }
    runProjectSchedule {
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
    user {
        ...User_nav
    }
    you {
        canDelete
        canEdit
        canView
    }
  }
}`;

export const runProjectDeleteAll = gql`
mutation runProjectDeleteAll {
  runProjectDeleteAll {
    count
  }
}`;

export const runProjectComplete = gql`${Organization_nav}
${User_nav}
${Label_full}

mutation runProjectComplete($input: RunProjectCompleteInput!) {
  runProjectComplete(input: $input) {
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
        directory {
            id
            created_at
            updated_at
            childOrder
            isRoot
            projectVersion {
                id
                isLatest
                isPrivate
                versionIndex
                versionLabel
                root {
                    id
                    isPrivate
                }
                translations {
                    id
                    language
                    description
                    name
                }
            }
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
    wasRunAutomaticaly
    organization {
        ...Organization_nav
    }
    projectVersion {
        id
        isLatest
        isPrivate
        versionIndex
        versionLabel
        root {
            id
            isPrivate
        }
        translations {
            id
            language
            description
            name
        }
    }
    runProjectSchedule {
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
    user {
        ...User_nav
    }
    you {
        canDelete
        canEdit
        canView
    }
  }
}`;

export const runProjectCancel = gql`${Organization_nav}
${User_nav}
${Label_full}

mutation runProjectCancel($input: RunProjectCancelInput!) {
  runProjectCancel(input: $input) {
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
        directory {
            id
            created_at
            updated_at
            childOrder
            isRoot
            projectVersion {
                id
                isLatest
                isPrivate
                versionIndex
                versionLabel
                root {
                    id
                    isPrivate
                }
                translations {
                    id
                    language
                    description
                    name
                }
            }
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
    wasRunAutomaticaly
    organization {
        ...Organization_nav
    }
    projectVersion {
        id
        isLatest
        isPrivate
        versionIndex
        versionLabel
        root {
            id
            isPrivate
        }
        translations {
            id
            language
            description
            name
        }
    }
    runProjectSchedule {
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
    user {
        ...User_nav
    }
    you {
        canDelete
        canEdit
        canView
    }
  }
}`;

