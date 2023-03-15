import * as yup from 'yup';
import { blankToUndefined, bool, description, id, jsonVariable, maxStrErr, name, opt, req, transRel, versionLabel, versionNotes, YupModel, yupObj } from "../utils";
import { resourceListValidation } from './resourceList';
import { smartContractValidation } from './smartContract';

const smartContractDefault = yup.string().transform(blankToUndefined).max(2048, maxStrErr);
const contractType = yup.string().transform(blankToUndefined).max(256, maxStrErr);
const content = yup.string().transform(blankToUndefined).max(8192, maxStrErr);

export const smartContractVersionTranslationValidation: YupModel = transRel({
    create: {
        description: opt(description),
        jsonVariable: opt(jsonVariable),
        name: req(name),
    },
    update: {
        description: opt(description),
        jsonVariable: opt(jsonVariable),
        name: opt(name),
    }
})

export const smartContractVersionValidation: YupModel = {
    create: ({ o, minVersion = '0.0.1' }) => yupObj({
        id: req(id),
        isComplete: opt(bool),
        isPrivate: opt(bool),
        default: opt(smartContractDefault),
        contractType: req(contractType),
        content: req(content),
        versionLabel: req(versionLabel(minVersion)),
        versionNotes: opt(versionNotes),
    }, [
        ['directoryListings', ['Connect'], 'many', 'opt'],
        ['resourceList', ['Create'], 'one', 'opt', resourceListValidation],
        ['root', ['Connect', 'Create'], 'one', 'req', smartContractValidation, ['versions']],
        ['translations', ['Create'], 'many', 'opt', smartContractVersionTranslationValidation],
    ], [['rootConnect', 'rootCreate']], o),
    update: ({ o, minVersion = '0.0.1' }) => yupObj({
        id: req(id),
        isComplete: opt(bool),
        isPrivate: opt(bool),
        default: opt(smartContractDefault),
        contractType: opt(contractType),
        content: opt(content),
        versionLabel: opt(versionLabel(minVersion)),
        versionNotes: opt(versionNotes),
    }, [
        ['directoryListings', ['Connect', 'Disconnect'], 'many', 'opt'],
        ['resourceList', ['Create', 'Update'], 'one', 'opt', resourceListValidation],
        ['root', ['Update'], 'one', 'opt', smartContractValidation, ['versions']],
        ['translations', ['Create', 'Update', 'Delete'], 'many', 'opt', smartContractVersionTranslationValidation],
    ], [], o),
}