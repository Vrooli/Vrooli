import gql from 'graphql-tag';
import { Api_list } from '../fragments/Api_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';
import { Tag_list } from '../fragments/Tag_list';
import { Label_list } from '../fragments/Label_list';
import { NoteVersion_list } from '../fragments/NoteVersion_list';
import { Organization_list } from '../fragments/Organization_list';
import { ProjectVersion_list } from '../fragments/ProjectVersion_list';
import { RoutineVersion_list } from '../fragments/RoutineVersion_list';
import { Label_full } from '../fragments/Label_full';
import { SmartContractVersion_list } from '../fragments/SmartContractVersion_list';
import { StandardVersion_list } from '../fragments/StandardVersion_list';

export const copyCopy = gql`${Api_list}
${Organization_nav}
${User_nav}
${Tag_list}
${Label_list}
${NoteVersion_list}
${Organization_list}
${ProjectVersion_list}
${RoutineVersion_list}
${Label_full}
${SmartContractVersion_list}
${StandardVersion_list}

mutation copy($input: CopyInput!) {
  copy(input: $input) {
    apiVersion {
        ...Api_list
    }
    noteVersion {
        ...NoteVersion_list
    }
    organization {
        ...Organization_list
    }
    projectVersion {
        ...ProjectVersion_list
    }
    routineVersion {
        ...RoutineVersion_list
    }
    smartContractVersion {
        ...SmartContractVersion_list
    }
    standardVersion {
        ...StandardVersion_list
    }
  }
}`;

