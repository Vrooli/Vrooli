import gql from "graphql-tag";
import { Api_list } from "../fragments/Api_list";
import { Api_nav } from "../fragments/Api_nav";
import { Label_full } from "../fragments/Label_full";
import { Label_list } from "../fragments/Label_list";
import { Note_list } from "../fragments/Note_list";
import { Note_nav } from "../fragments/Note_nav";
import { Organization_list } from "../fragments/Organization_list";
import { Organization_nav } from "../fragments/Organization_nav";
import { Project_list } from "../fragments/Project_list";
import { Project_nav } from "../fragments/Project_nav";
import { Question_list } from "../fragments/Question_list";
import { Routine_list } from "../fragments/Routine_list";
import { Routine_nav } from "../fragments/Routine_nav";
import { SmartContract_list } from "../fragments/SmartContract_list";
import { SmartContract_nav } from "../fragments/SmartContract_nav";
import { Standard_list } from "../fragments/Standard_list";
import { Standard_nav } from "../fragments/Standard_nav";
import { Tag_list } from "../fragments/Tag_list";
import { User_list } from "../fragments/User_list";
import { User_nav } from "../fragments/User_nav";

export const feedPopular = gql`${Api_list}
${Api_nav}
${Label_full}
${Label_list}
${Note_list}
${Note_nav}
${Organization_list}
${Organization_nav}
${Project_list}
${Project_nav}
${Question_list}
${Routine_list}
${Routine_nav}
${SmartContract_list}
${SmartContract_nav}
${Standard_list}
${Standard_nav}
${Tag_list}
${User_list}
${User_nav}

query popular($input: PopularInput!) {
  popular(input: $input) {
    apis {
        ...Api_list
    }
    notes {
        ...Note_list
    }
    organizations {
        ...Organization_list
    }
    projects {
        ...Project_list
    }
    questions {
        ...Question_list
    }
    routines {
        ...Routine_list
    }
    smartContracts {
        ...SmartContract_list
    }
    standards {
        ...Standard_list
    }
    users {
        ...User_list
    }
  }
}`;

