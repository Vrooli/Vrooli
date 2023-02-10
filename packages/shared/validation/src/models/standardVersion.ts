import * as yup from 'yup';
import { InputType } from "@shared/consts";
import { blankToUndefined, bool, description, enumToYup, id, jsonVariable, maxStrErr, opt, req, transRel, versionLabel, versionNotes, YupModel, yupObj } from "../utils";
import { resourceListValidation } from "./resourceList";
import { standardValidation } from "./standard";

const standardDefault = yup.string().transform(blankToUndefined).max(2048, maxStrErr);
const standardType = enumToYup(InputType);
const standardProps = yup.string().transform(blankToUndefined).max(8192, maxStrErr);
const standardYup = yup.string().transform(blankToUndefined).max(8192, maxStrErr);

export const standardVersionTranslationValidation: YupModel = transRel({
    create: {
        description: opt(description),
        jsonVariable: opt(jsonVariable),
    },
    update: {
        description: opt(description),
        jsonVariable: opt(jsonVariable),
    }
})

export const standardVersionValidation: YupModel = {
    create: ({ o, minVersion = '0.0.1' }) => yupObj({
        id: req(id),
        isComplete: opt(bool),
        isFile: opt(bool),
        isLatest: opt(bool),
        isPrivate: opt(bool),
        default: opt(standardDefault),
        standardType: req(standardType),
        props: req(standardProps),
        yup: opt(standardYup),
        versionLabel: req(versionLabel(minVersion)),
        versionNotes: opt(versionNotes),
    }, [
        ['directoryListings', ['Connect'], 'many', 'opt'],
        ['resourceList', ['Create'], 'one', 'opt', resourceListValidation],
        ['root', ['Connect', 'Create'], 'one', 'req', standardValidation, ['versions']],
        ['translations', ['Create'], 'many', 'opt', standardVersionTranslationValidation],
    ], [['rootConnect', 'rootCreate']], o),
    update: ({ o, minVersion = '0.0.1' }) => yupObj({
        id: req(id),
        isComplete: opt(bool),
        isFile: opt(bool),
        isLatest: opt(bool),
        isPrivate: opt(bool),
        default: opt(standardDefault),
        standardType: opt(standardType),
        props: opt(standardProps),
        yup: opt(standardYup),
        versionLabel: opt(versionLabel(minVersion)),
        versionNotes: opt(versionNotes),
    }, [
        ['directoryListings', ['Connect', 'Disconnect'], 'many', 'opt'],
        ['resourceList', ['Create', 'Update'], 'one', 'opt', resourceListValidation],
        ['root', ['Update'], 'one', 'opt', standardValidation, ['versions']],
        ['translations', ['Create', 'Update', 'Delete'], 'many', 'opt', standardVersionTranslationValidation],
    ], [], o),
}