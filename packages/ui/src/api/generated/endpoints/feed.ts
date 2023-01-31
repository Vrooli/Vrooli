import gql from 'graphql-tag';
import { Project_list } from '../fragments/Project_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';
import { Tag_list } from '../fragments/Tag_list';
import { Label_list } from '../fragments/Label_list';
import { Routine_list } from '../fragments/Routine_list';
import { Label_full } from '../fragments/Label_full';

export const feedPopular = gql`...${Organization_list}
...${Tag_list}
...${Project_list}
...${Organization_nav}
...${User_nav}
...${Label_list}
...${Routine_list}
...${Label_full}
...${Standard_list}
...${User_list}

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

export const feedLearn = gql`...${Project_list}
...${Organization_nav}
...${User_nav}
...${Tag_list}
...${Label_list}
...${Routine_list}
...${Label_full}

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

export const feedResearch = gql`...${Routine_list}
...${Organization_nav}
...${User_nav}
...${Label_full}
...${Tag_list}
...${Label_list}
...${Project_list}
...${Organization_list}

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

export const feedDevelop = gql`...${Project_list}
...${Organization_nav}
...${User_nav}
...${Tag_list}
...${Label_list}
...${Routine_list}
...${Label_full}

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

