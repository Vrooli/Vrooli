import gql from 'graphql-tag';
import { Label_list } from '../fragments/Label_list';
import { Meeting_list } from '../fragments/Meeting_list';
import { Note_list } from '../fragments/Note_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { Reminder_full } from '../fragments/Reminder_full';
import { Resource_list } from '../fragments/Resource_list';
import { RunProjectSchedule_list } from '../fragments/RunProjectSchedule_list';
import { RunRoutineSchedule_list } from '../fragments/RunRoutineSchedule_list';
import { Tag_list } from '../fragments/Tag_list';
import { User_nav } from '../fragments/User_nav';

export const feedHome = gql`${Label_list}
${Meeting_list}
${Note_list}
${Organization_nav}
${Reminder_full}
${Resource_list}
${RunProjectSchedule_list}
${RunRoutineSchedule_list}
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
    schedules {
        ...RunProjectSchedule_list
    }
    resources {
        ...Resource_list
    }
    runRoutineSchedules {
        ...RunRoutineSchedule_list
    }
  }
}`;

