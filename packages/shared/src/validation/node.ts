/**
 * Nodes are a bit more complicated than the other types because they come in
 * several variations. Nonetheless, they are handled in the same way - connect, 
 * disconnect, delete, create, and update
 */
import { description, idArray, id, title } from './base';
import * as yup from 'yup';

const columnIndex = yup.number().integer().min(0).optional();
const rowIndex = yup.number().integer().min(0).optional();
export const condition = yup.string().max(2048).optional();
const isOptional = yup.boolean().optional();
const loops = yup.number().integer().min(0).max(100).optional();
const maxLoops = yup.number().integer().min(1).max(100).optional();
const type = yup.string().oneOf(["End", "Loop", "RoutineList", "Redirect", "Start"]).optional();
const wasSuccessful = yup.boolean().optional();

export const nodeEndCreate = yup.object().shape({
    wasSuccessful,
})
export const nodeEndUpdate = yup.object().shape({
    id: id.required(),
    wasSuccessful,
})

export const conditionWhenCreate = yup.array().of(yup.object().shape({
    condition: condition.required(),
}).required()).optional();
export const conditionWhenUpdate = yup.array().of(yup.object().shape({
    id: id.required(),
    condition: condition.required(),
}).required()).optional();
export const conditionsCreate = yup.array().of(yup.object().shape({
    description,
    title: title.required(),
    whenCreate: conditionWhenCreate,
    toId: id,
}).required()).optional();
export const conditionsUpdate = yup.array().of(yup.object().shape({
    id: id.required(),
    description,
    title,
    whenCreate: conditionWhenCreate,
    whenUpdate: conditionWhenUpdate,
    whenDelete: idArray,
}).required()).optional();
export const nodeLinkCreate = yup.object().shape({
    conditionsCreate,
    fromId: id.required(),
    toId: id.required(),
})
export const nodeLinkUpdate = yup.object().shape({
    id: id.required(),
    conditionsCreate,
    conditionsUpdate,
    conditionsDelete: idArray,
})
export const nodeLinksCreate = yup.array().of(nodeLinkCreate.required()).optional();
export const nodeLinksUpdate = yup.array().of(nodeLinkUpdate.required()).optional();

export const loopWhenCreate = yup.array().of(yup.object().shape({
    condition: condition.required(),
}).required()).optional();
export const loopWhenUpdate = yup.array().of(yup.object().shape({
    id: id.required(),
    condition: condition.required(),
}).required()).optional();
export const whilesCreate = yup.array().of(yup.object().shape({
    description,
    title: title.required(),
    whenCreate: loopWhenCreate,
    toId: id,
}).required()).optional();
export const whilesUpdate = yup.array().of(yup.object().shape({
    id: id.required(),
    description,
    title,
    toId: id,
    whenCreate: loopWhenCreate,
    whenUpdate: loopWhenUpdate,
    whenDelete: idArray,
}).required()).optional();
export const nodeLoopCreate = yup.object().shape({
    loops,
    maxLoops,
    whilesCreate: whilesCreate.required(),
})
export const nodeLoopUpdate = yup.object().shape({
    id: id.required(),
    loops,
    maxLoops,
    whilesCreate,
    whilesUpdate,
    whilesDelete: idArray,
})

export const nodeRoutineListItemsCreate = yup.array().of(yup.object().shape({
    description,
    title,
    isOptional,
    // Cannot create a routine directly from a node, as this causes a cyclic dependency
    routineConnect: id,
}).required()).optional();
export const nodeRoutineListItemsUpdate = yup.array().of(yup.object().shape({
    id: id.required(),
    description,
    isOptional,
    title,
    // Cannot create or update a routine directly from a node, as this causes a cyclic dependency
    routineConnect: id,
    routineDisconnect: id,
    routineDelete: id,
}).required()).optional();
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

export const nodeCreate = yup.object().shape({
    columnIndex,
    description,
    rowIndex,
    title: title.required(),
    type: type.required(),
    nodeEndCreate,
    nodeLoopCreate,
    nodeRoutineListCreate,
    routineId: id.required(),
})
/**
 * A node always contains one data type. Since this is known, there is no need 
 * to specify a disconnect or delete
 */
export const nodeUpdate = yup.object().shape({
    id,
    columnIndex,
    description,
    rowIndex,
    title,
    type,
    nodeEndCreate,
    nodeEndUpdate,
    nodeLoopCreate,
    nodeLoopUpdate,
    nodeRoutineListCreate,
    nodeRoutineListUpdate,
    routineId: id,
})

export const nodesCreate = yup.array().of(nodeCreate.required()).optional();
export const nodesUpdate = yup.array().of(nodeUpdate.required()).optional();