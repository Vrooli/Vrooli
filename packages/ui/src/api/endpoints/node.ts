import { nodeFields as fullFields } from 'api/partial';
import { toMutation} from 'api/utils';

export const nodeEndpoint = {
    create: toMutation('nodeCreate', 'NodeCreateInput', fullFields[1]),
    update: toMutation('nodeUpdate', 'NodeUpdateInput', fullFields[1])
}