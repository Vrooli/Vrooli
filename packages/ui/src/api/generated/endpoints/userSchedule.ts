import gql from 'graphql-tag';
import { Label_full } from '../fragments/Label_full';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';
import { Label_list } from '../fragments/Label_list';

export const userScheduleFindOne = gql`...${Label_full}
...${Organization_nav}
...${User_nav}
...${Label_list}

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

export const userScheduleFindMany = gql`...${Label_list}
...${Organization_nav}
...${User_nav}

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

export const userScheduleCreate = gql`...${Label_full}
...${Organization_nav}
...${User_nav}
...${Label_list}

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

export const userScheduleUpdate = gql`...${Label_full}
...${Organization_nav}
...${User_nav}
...${Label_list}

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

