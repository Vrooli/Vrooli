import * as yup from 'yup';
import { apiCallData, description, id, index, instructions, name, opt, rel, req, smartContractCallData, transRel, versionLabel, versionNotes, YupModel } from "../utils";
import { nodeValidation } from './node';
import { nodeLinkValidation } from './nodeLink';
import { resourceListValidation } from './resourceList';
import { routineValidation } from './routine';
import { routineVersionInputValidation } from './routineVersionInput';
import { routineVersionOutputValidation } from './routineVersionOutput';

export const routineVersionTranslationValidation: YupModel = transRel({
    create: {
        description: opt(description),
        instructions: opt(instructions),
        name: req(name),
    },
    update: {
        description: opt(description),
        instructions: opt(instructions),
        name: opt(name),
    }
})

export const routineVersionValidation: YupModel = {
    create: () => yup.object().shape({
        id: req(id),
        apiCallData: opt(apiCallData),
        smartContractCallData: opt(smartContractCallData),
        isComplete: opt(yup.boolean()),
        isInternal: opt(yup.boolean()),
        isLatest: opt(yup.boolean()),
        isPrivate: opt(yup.boolean()),
        versionIndex: req(index),
        versionLabel: req(versionLabel('0.0.1')),
        versionNotes: opt(versionNotes),
        ...rel('root', ['Connect', 'Create'], 'one', 'req', routineValidation),
        ...rel('apiVersion', ['Connect'], 'one', 'req'),
        ...rel('smartContractVersion', ['Connect'], 'one', 'req'),
        ...rel('resourceList', ['Create'], 'one', 'req', resourceListValidation),
        ...rel('nodes', ['Create'], 'many', 'req', nodeValidation),
        ...rel('nodeLinks', ['Create'], 'many', 'req', nodeLinkValidation),
        ...rel('inputs', ['Create'], 'many', 'req', routineVersionInputValidation),
        ...rel('outputs', ['Create'], 'many', 'req', routineVersionOutputValidation),
        ...rel('translations', ['Create'], 'many', 'opt', routineVersionTranslationValidation),
        ...rel('directoryListings', ['Connect'], 'many', 'opt'),
        ...rel('suggestedNextByProject', ['Connect'], 'many', 'opt'),
    }, [['rootConnect', 'rootCreate']]),
    update: () => yup.object().shape({
        id: req(id),
        apiCallData: opt(apiCallData),
        smartContractCallData: opt(smartContractCallData),
        isComplete: opt(yup.boolean()),
        isInternal: opt(yup.boolean()),
        isLatest: opt(yup.boolean()),
        isPrivate: opt(yup.boolean()),
        versionIndex: opt(index),
        versionLabel: opt(versionLabel('0.0.1')),
        versionNotes: opt(versionNotes),
        ...rel('apiVersion', ['Connect', 'Disconnect'], 'one', 'req'),
        ...rel('smartContractVersion', ['Connect', 'Disconnect'], 'one', 'req'),
        ...rel('resourceList', ['Create', 'Update'], 'one', 'req', resourceListValidation),
        ...rel('nodes', ['Create', 'Update', 'Delete'], 'many', 'req', nodeValidation),
        ...rel('nodeLinks', ['Create', 'Update', 'Delete'], 'many', 'req', nodeLinkValidation),
        ...rel('inputs', ['Create', 'Update', 'Delete'], 'many', 'req', routineVersionInputValidation),
        ...rel('outputs', ['Create', 'Update', 'Delete'], 'many', 'req', routineVersionOutputValidation),
        ...rel('translations', ['Create', 'Update', 'Delete'], 'many', 'opt', routineVersionTranslationValidation),
        ...rel('directoryListings', ['Connect', 'Disconnect'], 'many', 'opt'),
        ...rel('suggestedNextByProject', ['Connect', 'Disconnect'], 'many', 'opt'),
    }),
}