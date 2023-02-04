import gql from 'graphql-tag';
import { Label_full } from '../fragments/Label_full';
import { Label_list } from '../fragments/Label_list';
import { Organization_list } from '../fragments/Organization_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { Project_list } from '../fragments/Project_list';
import { Routine_list } from '../fragments/Routine_list';
import { Standard_list } from '../fragments/Standard_list';
import { Tag_list } from '../fragments/Tag_list';
import { User_list } from '../fragments/User_list';
import { User_nav } from '../fragments/User_nav';

export const feedPopular = gql`${Label_full}
${Label_list}
${Organization_list}
${Organization_nav}
${Project_list}
${Routine_list}
${Standard_list}
${Tag_list}
${User_list}
${User_nav}

query popular($input: PopularInput!) {
  popular(input: $input) {
    organizations {
        ...Organization_list
    }
    projects {
        ...Project_list
    }
    routines {
        ...Routine_list
    }
    standards {
        ...Standard_list
    }
    users {
        ...User_list
    }
  }
}`;

export const feedLearn = gql`${Label_full}
${Label_list}
${Organization_nav}
${Project_list}
${Routine_list}
${Tag_list}
${User_nav}

query learn {
  learn {
    courses {
        ...Project_list
    }
    tutorials {
        ...Routine_list
    }
  }
}`;

export const feedResearch = gql`${Label_full}
${Label_list}
${Organization_list}
${Organization_nav}
${Project_list}
${Routine_list}
${Tag_list}
${User_nav}

query research {
  research {
    processes {
        ...Routine_list
    }
    newlyCompleted {
        ... on Routine {
            ...Routine_list
        }
        ... on Project {
            ...Project_list
        }
    }
    needVotes {
        ...Project_list
    }
    needInvestments {
        ...Project_list
    }
    needMembers {
        ...Organization_list
    }
  }
}`;

export const feedDevelop = gql`${Label_full}
${Label_list}
${Organization_nav}
${Project_list}
${Routine_list}
${Tag_list}
${User_nav}

query develop {
  develop {
    completed {
        ... on Project {
            ...Project_list
        }
        ... on Routine {
            ...Routine_list
        }
    }
    inProgress {
        ... on Project {
            ...Project_list
        }
        ... on Routine {
            ...Routine_list
        }
    }
    recent {
        ... on Project {
            ...Project_list
        }
        ... on Routine {
            ...Routine_list
        }
    }
  }
}`;

