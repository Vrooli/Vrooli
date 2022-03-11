/**
 * Nodes are a bit more complicated than the other types because they come in
 * several variations. Nonetheless, they are handled in the same way - connect, 
 * disconnect, delete, create, and update
 */
import { description, idArray, id, title, language } from './base';
import * as yup from 'yup';

const columnIndex = yup.number().integer().min(0).nullable()
const rowIndex = yup.number().integer().min(0).nullable()
export const condition = yup.string().max(8192)
const isOptional = yup.boolean()
const loops = yup.number().integer().min(0).max(100)
const maxLoops = yup.number().integer().min(1).max(100)
const operation = yup.string().max(512)
const type = yup.string().oneOf(["End", "Loop", "RoutineList", "Redirect", "Start"])
const wasSuccessful = yup.boolean()

export const nodeEndCreate = yup.object().shape({
    wasSuccessful: wasSuccessful.notRequired().default(undefined),
})
export const nodeEndUpdate = yup.object().shape({
    id: id.required(),
    wasSuccessful: wasSuccessful.notRequired().default(undefined),
})

export const whenTranslationCreate = yup.object().shape({
    language: language.required(),
    description: description.notRequired().default(undefined),
    title: title.required(),
});
export const whenTranslationUpdate = yup.object().shape({
    id: id.required(),
    language: language.notRequired().default(undefined),
    description: description.notRequired().default(undefined),
    title: title.notRequired().default(undefined),
});
const whenTranslationsCreate = yup.array().of(whenTranslationCreate.required())
const whenTranslationsUpdate = yup.array().of(whenTranslationUpdate.required())
export const whenCreate = yup.object().shape({
    condition: condition.required(),
    translationsCreate: whenTranslationsCreate.notRequired().default(undefined),
    translationsUpdate: whenTranslationsUpdate.notRequired().default(undefined),
    translationsDelete: idArray.notRequired().default(undefined),
});
export const whenUpdate = yup.object().shape({
    id: id.required(),
    condition: condition.notRequired().default(undefined),
    translationsCreate: whenTranslationsCreate.notRequired().default(undefined),
    translationsUpdate: whenTranslationsUpdate.notRequired().default(undefined),
    translationsDelete: idArray.notRequired().default(undefined),
});
export const whensCreate = yup.array().of(whenCreate.required())
export const whensUpdate = yup.array().of(whenUpdate.required())

export const nodeLinkCreate = yup.object().shape({
    fromId: id.required(),
    toId: id.required(),
    operation: operation.notRequired().default(undefined),
    whensCreate: whensCreate.notRequired().default(undefined),
})
export const nodeLinkUpdate = yup.object().shape({
    id: id.required(),
    operation: operation.notRequired().default(undefined),
    whensCreate: whensCreate.notRequired().default(undefined),
    whensUpdate: whensUpdate.notRequired().default(undefined),
    whensDelete: idArray,
})
export const nodeLinksCreate = yup.array().of(nodeLinkCreate.required())
export const nodeLinksUpdate = yup.array().of(nodeLinkUpdate.required())

export const whileTranslationCreate = yup.object().shape({
    language: language.required(),
    description: description.notRequired().default(undefined),
    title: title.required(),
});
export const whileTranslationUpdate = yup.object().shape({
    id: id.required(),
    language: language.notRequired().default(undefined),
    description: description.notRequired().default(undefined),
    title: title.notRequired().default(undefined),
});
export const whileTranslationsCreate = yup.array().of(whileTranslationCreate.required())
export const whileTranslationsUpdate = yup.array().of(whileTranslationUpdate.required())
export const whileCreate = yup.object().shape({
    condition: condition.required(),
    translationsCreate: whileTranslationsCreate.notRequired().default(undefined),
});
export const whileUpdate = yup.object().shape({
    id: id.required(),
    condition: condition.notRequired().default(undefined),
    whenDelete: idArray.notRequired().default(undefined),
    translationsCreate: whileTranslationsCreate.notRequired().default(undefined),
    translationsUpdate: whileTranslationsUpdate.notRequired().default(undefined),
    translationsDelete: idArray.notRequired().default(undefined),
});
export const whilesCreate = yup.array().of(whileCreate.required())
export const whilesUpdate = yup.array().of(whileUpdate.required())

export const loopCreate = yup.object().shape({
    loops: loops.notRequired().default(undefined),
    maxLoops: maxLoops.notRequired().default(undefined),
    operation: operation.notRequired().default(undefined),
    whilesCreate: whilesCreate.required(),
})
export const loopUpdate = yup.object().shape({
    id: id.required(),
    loops: loops.notRequired().default(undefined),
    maxLoops: maxLoops.notRequired().default(undefined),
    operation: operation.notRequired().default(undefined),
    whilesCreate: whilesCreate.notRequired().default(undefined),
    whilesUpdate: whilesUpdate.notRequired().default(undefined),
    whilesDelete: idArray.notRequired().default(undefined),
})

export const nodeRoutineListItemTranslationCreate = yup.object().shape({
    language: language.required(),
    description: description.notRequired().default(undefined),
    title: title.required(),
});
export const nodeRoutineListItemTranslationUpdate = yup.object().shape({
    id: id.required(),
    language: language.notRequired().default(undefined),
    description: description.notRequired().default(undefined),
    title: title.notRequired().default(undefined),
});
export const nodeRoutineListItemTranslationsCreate = yup.array().of(nodeRoutineListItemTranslationCreate.required())
export const nodeRoutineListItemTranslationsUpdate = yup.array().of(nodeRoutineListItemTranslationUpdate.required())
export const nodeRoutineListItemCreate = yup.object().shape({
    isOptional: isOptional.notRequired().default(undefined),
    routineId: id.required(),
    translationsCreate: nodeRoutineListItemTranslationsCreate.notRequired().default(undefined),
});
export const nodeRoutineListItemUpdate = yup.object().shape({
    id: id.required(),
    isOptional: isOptional.notRequired().default(undefined),
    routineId: id.notRequired().default(undefined),
    translationsDelete: idArray.notRequired().default(undefined),
    translationsCreate: nodeRoutineListItemTranslationsCreate.notRequired().default(undefined),
    translationsUpdate: nodeRoutineListItemTranslationsUpdate.notRequired().default(undefined),
});
export const nodeRoutineListItemsCreate = yup.array().of(nodeRoutineListItemCreate.required())
export const nodeRoutineListItemsUpdate = yup.array().of(nodeRoutineListItemUpdate.required())

export const nodeRoutineListCreate = yup.object().shape({
    isOrdered: yup.boolean().notRequired().default(undefined),
    isOptional: yup.boolean().notRequired().default(undefined),
    routinesConnect: idArray.notRequired().default(undefined),
    routinesCreate: nodeRoutineListItemsCreate.notRequired().default(undefined),
})
export const nodeRoutineListUpdate = yup.object().shape({
    id: id.required(),
    isOrdered: yup.boolean().notRequired().default(undefined),
    isOptional: yup.boolean().notRequired().default(undefined),
    routinesConnect: idArray.notRequired().default(undefined),
    routinesDisconnect: idArray.notRequired().default(undefined),
    routinesDelete: idArray.notRequired().default(undefined),
    routinesCreate: nodeRoutineListItemsCreate.notRequired().default(undefined),
    routinesUpdate: nodeRoutineListItemsUpdate.notRequired().default(undefined),
})

export const nodeTranslationCreate = yup.object().shape({
    language: language.required(),
    description: description.notRequired().default(undefined),
    title: title.required(),
});
export const nodeTranslationUpdate = yup.object().shape({
    id: id.required(),
    language: language.notRequired().default(undefined),
    description: description.notRequired().default(undefined),
    title: title.notRequired().default(undefined),
});
export const nodeTranslationsCreate = yup.array().of(nodeTranslationCreate.required())
export const nodeTranslationsUpdate = yup.array().of(nodeTranslationUpdate.required())

export const nodeCreate = yup.object().shape({
    columnIndex: columnIndex.notRequired().default(undefined),
    rowIndex: rowIndex.notRequired().default(undefined),
    type: type.required(),
    nodeEndCreate: nodeEndCreate.notRequired().default(undefined),
    loopCreate: loopCreate.notRequired().default(undefined),
    nodeRoutineListCreate: nodeRoutineListCreate.notRequired().default(undefined),
    routineId: id.required(),
    translationsCreate: nodeTranslationsCreate.notRequired().default(undefined),
})
export const nodeUpdate = yup.object().shape({
    id: id.required(),
    columnIndex: columnIndex.notRequired().default(undefined),
    rowIndex: rowIndex.notRequired().default(undefined),
    type: type.notRequired().default(undefined),
    loopDelete: id.notRequired().default(undefined),
    loopCreate: loopCreate.notRequired().default(undefined),
    loopUpdate: loopUpdate.notRequired().default(undefined),
    nodeEndUpdate: nodeEndUpdate.notRequired().default(undefined),
    nodeRoutineListUpdate: nodeRoutineListUpdate.notRequired().default(undefined),
    routineId: id.notRequired().default(undefined),
    translationsDelete: idArray.notRequired().default(undefined),
    translationsCreate: nodeTranslationsCreate.notRequired().default(undefined),
    translationsUpdate: nodeTranslationsUpdate.notRequired().default(undefined),
})

export const nodesCreate = yup.array().of(nodeCreate.required())
export const nodesUpdate = yup.array().of(nodeUpdate.required())