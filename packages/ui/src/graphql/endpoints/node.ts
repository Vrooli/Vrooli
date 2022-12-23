import { nodeFields as fullFields } from 'graphql/partial';
import { toMutation} from 'graphql/utils';

export const nodeEndpoint = {
    create: toMutation('nodeCreate', 'NodeCreateInput', [fullFields], `...fullFields`),
    update: toMutation('nodeUpdate', 'NodeUpdateInput', [fullFields], `...fullFields`)
}