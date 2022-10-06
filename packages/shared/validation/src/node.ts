/**
 * Nodes are a bit more complicated than the other types because they come in
 * several variations. Nonetheless, they are handled in the same way - connect, 
 * disconnect, delete, create, and update
 */
import { description, idArray, id, title, language, maxStringErrorMessage, minStringErrorMessage, minNumberErrorMessage, requiredErrorMessage, blankToUndefined } from './base';
import * as yup from 'yup';

const index = yup.number().integer().min(0, minNumberErrorMessage).nullable();
const columnIndex = yup.number().integer().min(0, minNumberErrorMessage).nullable()
const rowIndex = yup.number().integer().min(0, minNumberErrorMessage).nullable()
export const condition = yup.string().transform(blankToUndefined).max(8192, maxStringErrorMessage)
const isOptional = yup.boolean()
const loops = yup.number().integer().min(0, minStringErrorMessage).max(100, maxStringErrorMessage)
const maxLoops = yup.number().integer().min(1, minStringErrorMessage).max(100, maxStringErrorMessage)
const operation = yup.string().transform(blankToUndefined).max(512, maxStringErrorMessage)
const type = yup.string().transform(blankToUndefined).oneOf(["End", "Loop", "RoutineList", "Redirect", "Start"])
const wasSuccessful = yup.boolean()

export const nodeEndCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    wasSuccessful: wasSuccessful.notRequired().default(undefined),
})
export const nodeEndUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    wasSuccessful: wasSuccessful.notRequired().default(undefined),
})
export const nodeEndForm = yup.object().shape({
    wasSuccessful: wasSuccessful.notRequired().default(undefined),
    title: title.required(requiredErrorMessage),
    description: description.notRequired().default(undefined),
});

export const whenTranslationCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    language: language.required(requiredErrorMessage),
    description: description.notRequired().default(undefined),
    title: title.required(requiredErrorMessage),
});
export const whenTranslationUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    language: language.notRequired().default(undefined),
    description: description.notRequired().default(undefined),
    title: title.notRequired().default(undefined),
});
const whenTranslationsCreate = yup.array().of(whenTranslationCreate.required(requiredErrorMessage))
const whenTranslationsUpdate = yup.array().of(whenTranslationUpdate.required(requiredErrorMessage))
export const whenCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    condition: condition.required(requiredErrorMessage),
    translationsCreate: whenTranslationsCreate.notRequired().default(undefined),
    translationsUpdate: whenTranslationsUpdate.notRequired().default(undefined),
    translationsDelete: idArray.notRequired().default(undefined),
});
export const whenUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    condition: condition.notRequired().default(undefined),
    translationsCreate: whenTranslationsCreate.notRequired().default(undefined),
    translationsUpdate: whenTranslationsUpdate.notRequired().default(undefined),
    translationsDelete: idArray.notRequired().default(undefined),
});
export const whensCreate = yup.array().of(whenCreate.required(requiredErrorMessage))
export const whensUpdate = yup.array().of(whenUpdate.required(requiredErrorMessage))

export const nodeLinkCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    fromId: id.required(requiredErrorMessage),
    toId: id.required(requiredErrorMessage),
    operation: operation.nullable().notRequired().default(undefined),
    whensCreate: whensCreate.notRequired().default(undefined),
})
export const nodeLinkUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    operation: operation.nullable().notRequired().default(undefined),
    whensCreate: whensCreate.notRequired().default(undefined),
    whensUpdate: whensUpdate.notRequired().default(undefined),
    whensDelete: idArray,
})
export const nodeLinksCreate = yup.array().of(nodeLinkCreate.required(requiredErrorMessage))
export const nodeLinksUpdate = yup.array().of(nodeLinkUpdate.required(requiredErrorMessage))

export const whileTranslationCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    language: language.required(requiredErrorMessage),
    description: description.notRequired().default(undefined),
    title: title.required(requiredErrorMessage),
});
export const whileTranslationUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    language: language.notRequired().default(undefined),
    description: description.notRequired().default(undefined),
    title: title.notRequired().default(undefined),
});
export const whileTranslationsCreate = yup.array().of(whileTranslationCreate.required(requiredErrorMessage))
export const whileTranslationsUpdate = yup.array().of(whileTranslationUpdate.required(requiredErrorMessage))
export const whileCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    condition: condition.required(requiredErrorMessage),
    translationsCreate: whileTranslationsCreate.notRequired().default(undefined),
});
export const whileUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    condition: condition.notRequired().default(undefined),
    whenDelete: idArray.notRequired().default(undefined),
    translationsCreate: whileTranslationsCreate.notRequired().default(undefined),
    translationsUpdate: whileTranslationsUpdate.notRequired().default(undefined),
    translationsDelete: idArray.notRequired().default(undefined),
});
export const whilesCreate = yup.array().of(whileCreate.required(requiredErrorMessage))
export const whilesUpdate = yup.array().of(whileUpdate.required(requiredErrorMessage))

export const loopCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    loops: loops.notRequired().default(undefined),
    maxLoops: maxLoops.notRequired().default(undefined),
    operation: operation.notRequired().default(undefined),
    whilesCreate: whilesCreate.required(requiredErrorMessage),
})
export const loopUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    loops: loops.notRequired().default(undefined),
    maxLoops: maxLoops.notRequired().default(undefined),
    operation: operation.notRequired().default(undefined),
    whilesCreate: whilesCreate.notRequired().default(undefined),
    whilesUpdate: whilesUpdate.notRequired().default(undefined),
    whilesDelete: idArray.notRequired().default(undefined),
})
export const loopsCreate = yup.array().of(loopCreate.required(requiredErrorMessage))
export const loopsUpdate = yup.array().of(loopUpdate.required(requiredErrorMessage))

export const nodeRoutineListItemTranslationCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    language: language.required(requiredErrorMessage),
    description: description.notRequired().default(undefined),
    title: title.required(requiredErrorMessage),
});
export const nodeRoutineListItemTranslationUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    language: language.notRequired().default(undefined),
    description: description.notRequired().default(undefined),
    title: title.notRequired().default(undefined),
});
export const nodeRoutineListItemTranslationsCreate = yup.array().of(nodeRoutineListItemTranslationCreate.required(requiredErrorMessage))
export const nodeRoutineListItemTranslationsUpdate = yup.array().of(nodeRoutineListItemTranslationUpdate.required(requiredErrorMessage))
export const nodeRoutineListItemCreate = yup.object().shape({
    id: id.notRequired().default(undefined),
    index: index.required(requiredErrorMessage),
    isOptional: isOptional.notRequired().default(undefined),
    routineConnect: id.notRequired().default(undefined), // Creating subroutines must be done in a separate request
    translationsCreate: nodeRoutineListItemTranslationsCreate.notRequired().default(undefined),
});
export const nodeRoutineListItemUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    index: index.notRequired().default(undefined),
    isOptional: isOptional.notRequired().default(undefined),
    routineConnect: id.notRequired().default(undefined), // Create/update/delete of subroutines must be done in a separate request
    routineDisconnect: id.notRequired().default(undefined),
    translationsDelete: idArray.notRequired().default(undefined),
    translationsCreate: nodeRoutineListItemTranslationsCreate.notRequired().default(undefined),
    translationsUpdate: nodeRoutineListItemTranslationsUpdate.notRequired().default(undefined),
});
export const nodeRoutineListItemsCreate = yup.array().of(nodeRoutineListItemCreate.required(requiredErrorMessage))
export const nodeRoutineListItemsUpdate = yup.array().of(nodeRoutineListItemUpdate.required(requiredErrorMessage))

export const nodeRoutineListCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    isOrdered: yup.boolean().notRequired().default(undefined),
    isOptional: yup.boolean().notRequired().default(undefined),
    routinesCreate: nodeRoutineListItemsCreate.notRequired().default(undefined),
})
export const nodeRoutineListUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    isOrdered: yup.boolean().notRequired().default(undefined),
    isOptional: yup.boolean().notRequired().default(undefined),
    routinesDelete: idArray.notRequired().default(undefined),
    routinesCreate: nodeRoutineListItemsCreate.notRequired().default(undefined),
    routinesUpdate: nodeRoutineListItemsUpdate.notRequired().default(undefined),
})

export const nodeTranslationCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    language: language.required(requiredErrorMessage),
    description: description.notRequired().default(undefined),
    title: title.required(requiredErrorMessage),
});
export const nodeTranslationUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    language: language.notRequired().default(undefined),
    description: description.notRequired().default(undefined),
    title: title.notRequired().default(undefined),
});
export const nodeTranslationsCreate = yup.array().of(nodeTranslationCreate.required(requiredErrorMessage))
export const nodeTranslationsUpdate = yup.array().of(nodeTranslationUpdate.required(requiredErrorMessage))

export const nodeCreate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    columnIndex: columnIndex.notRequired().default(undefined),
    rowIndex: rowIndex.notRequired().default(undefined),
    type: type.required(requiredErrorMessage),
    nodeEndCreate: nodeEndCreate.notRequired().default(undefined),
    loopCreate: loopCreate.notRequired().default(undefined),
    nodeRoutineListCreate: nodeRoutineListCreate.notRequired().default(undefined),
    routineId: id.notRequired().default(undefined), // Not required because it can be derived from the parent routine
    translationsCreate: nodeTranslationsCreate.notRequired().default(undefined),
})
export const nodeUpdate = yup.object().shape({
    id: id.required(requiredErrorMessage),
    columnIndex: columnIndex.notRequired().default(undefined),
    rowIndex: rowIndex.notRequired().default(undefined),
    type: type.notRequired().default(undefined),
    loopDelete: id.notRequired().default(undefined),
    loopCreate: loopCreate.notRequired().default(undefined),
    loopUpdate: loopUpdate.notRequired().default(undefined),
    nodeEndCreate: nodeEndCreate.notRequired().default(undefined),
    nodeEndUpdate: nodeEndUpdate.notRequired().default(undefined),
    nodeRoutineListCreate: nodeRoutineListCreate.notRequired().default(undefined),
    nodeRoutineListUpdate: nodeRoutineListUpdate.notRequired().default(undefined),
    routineId: id.notRequired().default(undefined),
    translationsDelete: idArray.notRequired().default(undefined),
    translationsCreate: nodeTranslationsCreate.notRequired().default(undefined),
    translationsUpdate: nodeTranslationsUpdate.notRequired().default(undefined),
})

export const nodesCreate = yup.array().of(nodeCreate.required(requiredErrorMessage))
export const nodesUpdate = yup.array().of(nodeUpdate.required(requiredErrorMessage))