import { nodeFields as fullFields } from 'graphql/partial';
import { toMutation} from 'graphql/utils';

export const nodeEndpoint = {
    create: toMutation('nodeCreate', 'NodeCreateInput', fullFields[1]),
    update: toMutation('nodeUpdate', 'NodeUpdateInput', fullFields[1])
}