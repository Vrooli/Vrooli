import gql from 'graphql-tag';
import { Api_list } from '../fragments/Api_list';
import { Label_full } from '../fragments/Label_full';
import { Label_list } from '../fragments/Label_list';
import { NoteVersion_list } from '../fragments/NoteVersion_list';
import { Organization_list } from '../fragments/Organization_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { ProjectVersion_list } from '../fragments/ProjectVersion_list';
import { RoutineVersion_list } from '../fragments/RoutineVersion_list';
import { SmartContractVersion_list } from '../fragments/SmartContractVersion_list';
import { StandardVersion_list } from '../fragments/StandardVersion_list';
import { Tag_list } from '../fragments/Tag_list';
import { User_nav } from '../fragments/User_nav';

export const copyCopy = gql`${Api_list}
${Label_full}
${Label_list}
${NoteVersion_list}
${Organization_list}
${Organization_nav}
${ProjectVersion_list}
${RoutineVersion_list}
${SmartContractVersion_list}
${StandardVersion_list}
${Tag_list}
${User_nav}

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

