/**
 * Nodes are a bit more complicated than the other types because they come in
 * several variations. Nonetheless, they are handled in the same way - connect, 
 * disconnect, delete, create, and update
 */
import { description, idArray, id, name, language, maxStrErr, minStrErr, minNumErr, blankToUndefined, req, reqArr, opt } from './base';
import * as yup from 'yup';

const index = yup.number().integer().min(0, minNumErr).nullable();
const columnIndex = yup.number().integer().min(0, minNumErr).nullable()
const rowIndex = yup.number().integer().min(0, minNumErr).nullable()
export const condition = yup.string().transform(blankToUndefined).max(8192, maxStrErr)
const isOptional = yup.boolean()
const loops = yup.number().integer().min(0, minStrErr).max(100, maxStrErr)
const maxLoops = yup.number().integer().min(1, minStrErr).max(100, maxStrErr)
const operation = yup.string().transform(blankToUndefined).max(512, maxStrErr)
const type = yup.string().transform(blankToUndefined).oneOf(["End", "Loop", "RoutineList", "Redirect", "Start"])
const wasSuccessful = yup.boolean()

export const nodeEndCreate = yup.object().shape({
    id: req(id),
    wasSuccessful: opt(wasSuccessful),
})
export const nodeEndUpdate = yup.object().shape({
    id: req(id),
    wasSuccessful: opt(wasSuccessful),
})
export const nodeEndForm = yup.object().shape({
    wasSuccessful: opt(wasSuccessful),
    name: req(name),
    description: opt(description),
});

export const whenTranslationCreate = yup.object().shape({
    id: req(id),
    language: req(language),
    description: opt(description),
    name: req(name),
});
export const whenTranslationUpdate = yup.object().shape({
    id: req(id),
    language: opt(language),
    description: opt(description),
    name: opt(name),
});
const whenTranslationsCreate = reqArr(whenTranslationCreate)
const whenTranslationsUpdate = reqArr(whenTranslationUpdate)
export const whenCreate = yup.object().shape({
    id: req(id),
    condition: req(condition),
    translationsCreate: opt(whenTranslationsCreate),
    translationsUpdate: opt(whenTranslationsUpdate),
    translationsDelete: opt(idArray),
});
export const whenUpdate = yup.object().shape({
    id: req(id),
    condition: opt(condition),
    translationsCreate: opt(whenTranslationsCreate),
    translationsUpdate: opt(whenTranslationsUpdate),
    translationsDelete: opt(idArray),
});
export const whensCreate = reqArr(whenCreate)
export const whensUpdate = reqArr(whenUpdate)

export const nodeLinkCreate = yup.object().shape({
    id: req(id),
    fromId: req(id),
    toId: req(id),
    operation: opt(operation.nullable()),
    whensCreate: opt(whensCreate),
})
export const nodeLinkUpdate = yup.object().shape({
    id: req(id),
    operation: opt(operation.nullable()),
    whensCreate: opt(whensCreate),
    whensUpdate: opt(whensUpdate),
    whensDelete: opt(idArray),
})
export const nodeLinksCreate = reqArr(nodeLinkCreate)
export const nodeLinksUpdate = reqArr(nodeLinkUpdate)

export const whileTranslationCreate = yup.object().shape({
    id: req(id),
    language: req(language),
    description: opt(description),
    name: req(name),
});
export const whileTranslationUpdate = yup.object().shape({
    id: req(id),
    language: opt(language),
    description: opt(description),
    name: opt(name),
});
export const whileTranslationsCreate = reqArr(whileTranslationCreate)
export const whileTranslationsUpdate = reqArr(whileTranslationUpdate)
export const whileCreate = yup.object().shape({
    id: req(id),
    condition: req(condition),
    translationsCreate: opt(whileTranslationsCreate),
});
export const whileUpdate = yup.object().shape({
    id: req(id),
    condition: opt(condition),
    whenDelete: opt(idArray),
    translationsCreate: opt(whileTranslationsCreate),
    translationsUpdate: opt(whileTranslationsUpdate),
    translationsDelete: opt(idArray),
});
export const whilesCreate = reqArr(whileCreate)
export const whilesUpdate = reqArr(whileUpdate)

export const loopCreate = yup.object().shape({
    id: req(id),
    loops: opt(loops),
    maxLoops: opt(maxLoops),
    operation: opt(operation),
    whilesCreate: req(whilesCreate),
})
export const loopUpdate = yup.object().shape({
    id: req(id),
    loops: opt(loops),
    maxLoops: opt(maxLoops),
    operation: opt(operation),
    whilesCreate: opt(whilesCreate),
    whilesUpdate: opt(whilesUpdate),
    whilesDelete: opt(idArray),
})
export const loopsCreate = reqArr(loopCreate)
export const loopsUpdate = reqArr(loopUpdate)

export const nodeRoutineListItemTranslationCreate = yup.object().shape({
    id: req(id),
    language: req(language),
    description: opt(description),
    name: req(name),
});
export const nodeRoutineListItemTranslationUpdate = yup.object().shape({
    id: req(id),
    language: opt(language),
    description: opt(description),
    name: opt(name),
});
export const nodeRoutineListItemTranslationsCreate = reqArr(nodeRoutineListItemTranslationCreate)
export const nodeRoutineListItemTranslationsUpdate = reqArr(nodeRoutineListItemTranslationUpdate)
export const nodeRoutineListItemCreate = yup.object().shape({
    id: req(id),
    index: req(index),
    isOptional: opt(isOptional),
    routineConnect: opt(id), // Creating subroutines must be done in a separate request
    translationsCreate: opt(nodeRoutineListItemTranslationsCreate),
});
export const nodeRoutineListItemUpdate = yup.object().shape({
    id: req(id),
    index: opt(index),
    isOptional: opt(isOptional),
    routineConnect: opt(id), // Create/update/delete of subroutines must be done in a separate request
    routineDisconnect: opt(id),
    translationsDelete: opt(idArray),
    translationsCreate: opt(nodeRoutineListItemTranslationsCreate),
    translationsUpdate: opt(nodeRoutineListItemTranslationsUpdate),
});
export const nodeRoutineListItemsCreate = reqArr(nodeRoutineListItemCreate)
export const nodeRoutineListItemsUpdate = reqArr(nodeRoutineListItemUpdate)

export const nodeRoutineListCreate = yup.object().shape({
    id: req(id),
    isOrdered: opt(yup.boolean()),
    isOptional: opt(yup.boolean()),
    routinesCreate: opt(nodeRoutineListItemsCreate),
})
export const nodeRoutineListUpdate = yup.object().shape({
    id: req(id),
    isOrdered: opt(yup.boolean()),
    isOptional: opt(yup.boolean()),
    routinesDelete: opt(idArray),
    routinesCreate: opt(nodeRoutineListItemsCreate),
    routinesUpdate: opt(nodeRoutineListItemsUpdate),
})

export const nodeTranslationCreate = yup.object().shape({
    id: req(id),
    language: req(language),
    description: req(description),
    name: req(name),
});
export const nodeTranslationUpdate = yup.object().shape({
    id: req(id),
    language: opt(language),
    description: req(description),
    name: opt(name),
});
export const nodeTranslationsCreate = reqArr(nodeTranslationCreate)
export const nodeTranslationsUpdate = reqArr(nodeTranslationUpdate)

export const nodeCreate = yup.object().shape({
    id: req(id),
    columnIndex: opt(columnIndex),
    rowIndex: opt(rowIndex),
    type: req(type),
    nodeEndCreate: opt(nodeEndCreate),
    loopCreate: opt(loopCreate),
    nodeRoutineListCreate: opt(nodeRoutineListCreate),
    routineId: opt(id), // Not required because it can be derived from the parent routine
    translationsCreate: opt(nodeTranslationsCreate),
})
export const nodeUpdate = yup.object().shape({
    id: req(id),
    columnIndex: opt(columnIndex),
    rowIndex: opt(rowIndex),
    type: opt(type),
    loopDelete: opt(id),
    loopCreate: opt(loopCreate),
    loopUpdate: opt(loopUpdate),
    nodeEndCreate: opt(nodeEndCreate),
    nodeEndUpdate: opt(nodeEndUpdate),
    nodeRoutineListCreate: opt(nodeRoutineListCreate),
    nodeRoutineListUpdate: opt(nodeRoutineListUpdate),
    routineId: opt(id),
    translationsDelete: opt(idArray),
    translationsCreate: opt(nodeTranslationsCreate),
    translationsUpdate: opt(nodeTranslationsUpdate),
})

export const nodesCreate = reqArr(nodeCreate)
export const nodesUpdate = reqArr(nodeUpdate)