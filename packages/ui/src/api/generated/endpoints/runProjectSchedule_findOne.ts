import gql from 'graphql-tag';
import { Label_full } from '../fragments/Label_full';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';

export const runProjectScheduleFindOne = gql`${Label_full}
${Organization_nav}
${User_nav}

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

