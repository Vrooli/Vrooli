import gql from 'graphql-tag';
import { Label_full } from '../fragments/Label_full';
import { Label_list } from '../fragments/Label_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { Schedule_common } from '../fragments/Schedule_common';
import { User_nav } from '../fragments/User_nav';

export const focusModeFindOne = gql`${Label_full}
${Label_list}
${Organization_nav}
${Schedule_common}
${User_nav}

query focusMode($input: FindByIdInput!) {
  focusMode(input: $input) {
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
}`;

