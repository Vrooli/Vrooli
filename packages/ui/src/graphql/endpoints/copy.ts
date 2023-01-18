import { apiVersionFields, noteVersionFields, organizationFields, projectVersionFields, routineVersionFields, smartContractVersionFields, standardVersionFields } from 'graphql/partial';
import { toMutation } from 'graphql/utils';

export const copyEndpoint = {
    copy: toMutation('copy', 'CopyInput', `
        apiVersion {
            ...copy0
        }
        noteVersion {
            ...copy1
        }
        organization {
            ...copy2
        }
        projectVersion {
            ...copy3
        }
        routineVersion {
            ...copy4
        }
        smartContractVersion {
            ...copy5
        }
        standardVersion {
            ...copy6
        }
    `, [apiVersionFields, noteVersionFields, organizationFields, projectVersionFields, routineVersionFields, smartContractVersionFields, standardVersionFields]),
}