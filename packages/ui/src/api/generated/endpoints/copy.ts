import gql from 'graphql-tag';
import { Api_list } from '../fragments/Api_list';
import { NoteVersion_list } from '../fragments/NoteVersion_list';
import { Organization_list } from '../fragments/Organization_list';
import { ProjectVersion_list } from '../fragments/ProjectVersion_list';
import { RoutineVersion_list } from '../fragments/RoutineVersion_list';
import { SmartContractVersion_list } from '../fragments/SmartContractVersion_list';
import { StandardVersion_list } from '../fragments/StandardVersion_list';

export const copyCopy = gql`${Api_list}
${NoteVersion_list}
${Organization_list}
${ProjectVersion_list}
${RoutineVersion_list}
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

