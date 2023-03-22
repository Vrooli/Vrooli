import gql from 'graphql-tag';
import { Label_list } from '../fragments/Label_list';
import { Note_list } from '../fragments/Note_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { Reminder_full } from '../fragments/Reminder_full';
import { Resource_list } from '../fragments/Resource_list';
import { Schedule_common } from '../fragments/Schedule_common';
import { Schedule_list } from '../fragments/Schedule_list';
import { Tag_list } from '../fragments/Tag_list';
import { User_nav } from '../fragments/User_nav';

export const feedHome = gql`${Label_list}
${Note_list}
${Organization_nav}
${Reminder_full}
${Resource_list}
${Schedule_common}
${Schedule_list}
${Tag_list}
${User_nav}

query home($input: HomeInput!) {
  home(input: $input) {
    notes {
        ...Note_list
    }
    reminders {
        ...Reminder_full
    }
    resources {
        ...Resource_list
    }
    schedules {
        ...Schedule_list
    }
  }
}`;

