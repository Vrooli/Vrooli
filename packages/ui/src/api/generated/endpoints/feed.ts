import gql from 'graphql-tag';
import { Organization_list } from '../fragments/Organization_list';
import { Project_list } from '../fragments/Project_list';
import { Routine_list } from '../fragments/Routine_list';
import { Standard_list } from '../fragments/Standard_list';
import { User_list } from '../fragments/User_list';

export const feedPopular = gql`${Organization_list}
${Project_list}
${Routine_list}
${Standard_list}
${User_list}

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

export const feedLearn = gql`${Project_list}
${Routine_list}

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

export const feedResearch = gql`${Organization_list}
${Project_list}
${Routine_list}

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

export const feedDevelop = gql`${Project_list}
${Routine_list}

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

