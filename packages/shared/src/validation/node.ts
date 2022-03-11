/**
 * Nodes are a bit more complicated than the other types because they come in
 * several variations. Nonetheless, they are handled in the same way - connect, 
 * disconnect, delete, create, and update
 */
import { description, idArray, id, title, language } from './base';
import * as yup from 'yup';

const columnIndex = yup.number().integer().min(0).optional();
const rowIndex = yup.number().integer().min(0).optional();
export const condition = yup.string().max(8192).optional();
const isOptional = yup.boolean().optional();
const loops = yup.number().integer().min(0).max(100).optional();
const maxLoops = yup.number().integer().min(1).max(100).optional();
const operation = yup.string().max(512).optional();
const type = yup.string().oneOf(["End", "Loop", "RoutineList", "Redirect", "Start"]).optional();
const wasSuccessful = yup.boolean().optional();

export const nodeEndCreate = yup.object().shape({
    wasSuccessful,
})
export const nodeEndUpdate = yup.object().shape({
    id: id.required(),
    wasSuccessful,
})

export const whenTranslationCreate = yup.object().shape({
    language,
    description,
    title: title.required(),
});
export const whenTranslationUpdate = yup.object().shape({
    id: id.required(),
    language,
    description,
    title,
});
const whenTranslationsCreate = yup.array().of(whenTranslationCreate.required()).optional();
const whenTranslationsUpdate = yup.array().of(whenTranslationUpdate.required()).optional();
export const whenCreate = yup.object().shape({
    condition: condition.required(),
    translationsCreate: whenTranslationsCreate,
    translationsUpdate: whenTranslationsUpdate,
    translationsDelete: idArray,
});
export const whenUpdate = yup.object().shape({
    id: id.required(),
    condition,
    translationsCreate: whenTranslationsCreate,
    translationsUpdate: whenTranslationsUpdate,
    translationsDelete: idArray,
});
export const whensCreate = yup.array().of(whenCreate.required()).optional();
export const whensUpdate = yup.array().of(whenUpdate.required()).optional();

export const nodeLinkCreate = yup.object().shape({
    fromId: id.required(),
    toId: id.required(),
    operation,
    whensCreate,
})
export const nodeLinkUpdate = yup.object().shape({
    id: id.required(),
    operation,
    whensCreate,
    whensUpdate,
    whensDelete: idArray,
})
export const nodeLinksCreate = yup.array().of(nodeLinkCreate.required()).optional();
export const nodeLinksUpdate = yup.array().of(nodeLinkUpdate.required()).optional();

export const whileTranslationCreate = yup.object().shape({
    language,
    description,
    title: title.required(),
});
export const whileTranslationUpdate = yup.object().shape({
    id: id.required(),
    language,
    description,
    title,
});
export const whileTranslationsCreate = yup.array().of(whileTranslationCreate.required()).optional();
export const whileTranslationsUpdate = yup.array().of(whileTranslationUpdate.required()).optional();
export const whileCreate = yup.object().shape({
    condition: condition.required(),
    toId: id,
    translationsCreate: whileTranslationsCreate,
});
export const whileUpdate = yup.object().shape({
    id: id.required(),
    toId: id,
    condition,
    whenDelete: idArray,
    translationsCreate: whileTranslationsCreate,
    translationsUpdate: whileTranslationsUpdate,
    translationsDelete: idArray,
});
export const whilesCreate = yup.array().of(whileCreate.required()).optional();
export const whilesUpdate = yup.array().of(whileUpdate.required()).optional();

export const loopCreate = yup.object().shape({
    loops,
    maxLoops,
    operation,
    whilesCreate: whilesCreate.required(),
})
export const loopUpdate = yup.object().shape({
    id: id.required(),
    loops,
    maxLoops,
    operation,
    whilesCreate,
    whilesUpdate,
    whilesDelete: idArray,
})

export const nodeRoutineListItemTranslationCreate = yup.object().shape({
    language,
    description,
    title,
});
export const nodeRoutineListItemTranslationUpdate = yup.object().shape({
    id: id.required(),
    language,
    description,
    title,
});
export const nodeRoutineListItemTranslationsCreate = yup.array().of(nodeRoutineListItemTranslationCreate.required()).optional();
export const nodeRoutineListItemTranslationsUpdate = yup.array().of(nodeRoutineListItemTranslationUpdate.required()).optional();
export const nodeRoutineListItemCreate = yup.object().shape({
    isOptional,
    translationsCreate: nodeRoutineListItemTranslationsCreate,
});
export const nodeRoutineListItemUpdate = yup.object().shape({
    id: id.required(),
    isOptional,
    translationsDelete: idArray,
    translationsCreate: nodeRoutineListItemTranslationsCreate,
    translationsUpdate: nodeRoutineListItemTranslationsUpdate,
});
export const nodeRoutineListItemsCreate = yup.array().of(nodeRoutineListItemCreate.required()).optional();
export const nodeRoutineListItemsUpdate = yup.array().of(nodeRoutineListItemUpdate.required()).optional();

export const nodeRoutineListCreate = yup.object().shape({
    isOrdered: yup.boolean().optional(),
    isOptional: yup.boolean().optional(),
    routinesConnect: idArray,
    routinesCreate: nodeRoutineListItemsCreate,
})
export const nodeRoutineListUpdate = yup.object().shape({
    id: id.required(),
    isOrdered: yup.boolean().optional(),
    isOptional: yup.boolean().optional(),
    routinesConnect: idArray,
    routinesDisconnect: idArray,
    routinesDelete: idArray,
    routinesCreate: nodeRoutineListItemsCreate,
    routinesUpdate: nodeRoutineListItemsUpdate,
})

export const nodeTranslationCreate = yup.object().shape({
    language,
    description,
    title: title.required(),
});
export const nodeTranslationUpdate = yup.object().shape({
    id: id.required(),
    language,
    description,
    title,
});
export const nodeTranslationsCreate = yup.array().of(nodeTranslationCreate.required()).optional();
export const nodeTranslationsUpdate = yup.array().of(nodeTranslationUpdate.required()).optional();

export const nodeCreate = yup.object().shape({
    columnIndex,
    rowIndex,
    type: type.required(),
    nodeEndCreate,
    loopCreate,
    nodeRoutineListCreate,
    routineId: id.required(),
    translationsCreate: nodeTranslationsCreate,
})
/**
 * A node always contains one data type. Since this is known, there is no need 
 * to specify a disconnect or delete
 */
export const nodeUpdate = yup.object().shape({
    id,
    columnIndex,
    rowIndex,
    type,
    loopDelete: id,
    loopCreate,
    loopUpdate,
    nodeEndUpdate,
    nodeRoutineListUpdate,
    routineId: id,
    translationsDelete: idArray,
    translationsCreate: nodeTranslationsCreate,
    translationsUpdate: nodeTranslationsUpdate,
})

export const nodesCreate = yup.array().of(nodeCreate.required()).optional();
export const nodesUpdate = yup.array().of(nodeUpdate.required()).optional();