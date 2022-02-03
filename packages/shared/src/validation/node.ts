/**
 * Nodes are a bit more complicated than the other types because they come in
 * several variations. Nonetheless, they are handled in the same way - connect, 
 * disconnect, delete, add, and update
 */
import { description, idArray, id, title } from './base';
import * as yup from 'yup';
import { NodeType } from '../consts';

const condition = yup.string().max(2048).optional();
const isOptional = yup.boolean().optional();
const type = yup.string().oneOf(Object.values(NodeType)).optional();
const wasSuccessful = yup.boolean().optional();

const nodeCombineAdd = yup.object().shape({
    from: idArray.required(),
    toId: id,
})
const nodeCombineUpdate = yup.object().shape({
    id: id.required(),
    from: idArray,
    toId: id,
})

const whenAdd = yup.array().of(yup.object().shape({
    condition: condition.required(),
}).required()).optional();
const whenUpdate = yup.array().of(yup.object().shape({
    id: id.required(),
    condition: condition.required(),
}).required()).optional();
const decisionsAdd = yup.array().of(yup.object().shape({
    description,
    title: title.required(),
    whenAdd,
    toId: id,
}).required()).optional();
const decisionsUpdate = yup.array().of(yup.object().shape({
    id: id.required(),
    description,
    title,
    toId: id,
    whenAdd,
    whenUpdate,
    whenDelete: idArray,
}).required()).optional();
const nodeDecisionAdd = yup.object().shape({
    decisionsAdd: decisionsAdd.required(),
})
const nodeDecisionUpdate = yup.object().shape({
    id: id.required(),
    decisionsAdd,
    decisionsUpdate,
    decisionsDelete: idArray,
})

const nodeEndAdd = yup.object().shape({
    wasSuccessful,
})
const nodeEndUpdate = yup.object().shape({
    id: id.required(),
    wasSuccessful,
})

// TODO define
const nodeLoopAdd = yup.object().shape({
    id,
})
// TODO define
const nodeLoopUpdate = yup.object().shape({
    id: id.required(),
})

// TODO define
const nodeRedirectAdd = yup.object().shape({
    id,
})
// TODO define
const nodeRedirectUpdate = yup.object().shape({
    id: id.required(),
})

const nodeRoutineListItemsAdd = yup.array().of(yup.object().shape({
    description,
    title,
    isOptional,
    // Cannot add a routine directly from a node, as this causes a cyclic dependency
    routineConnect: id,
}).required()).optional();
const nodeRoutineListItemsUpdate = yup.array().of(yup.object().shape({
    id: id.required(),
    description,
    isOptional,
    title,
    // Cannot add or update a routine directly from a node, as this causes a cyclic dependency
    routineConnect: id,
    routineDisconnect: id,
    routineDelete: id,
}).required()).optional();
const nodeRoutineListAdd = yup.object().shape({
    isOrdered: yup.boolean().optional(),
    isOptional: yup.boolean().optional(),
    routinesConnect: idArray,
    routinesAdd: nodeRoutineListItemsAdd,
})
const nodeRoutineListUpdate = yup.object().shape({
    id: id.required(),
    isOrdered: yup.boolean().optional(),
    isOptional: yup.boolean().optional(),
    routinesConnect: idArray,
    routinesDisconnect: idArray,
    routinesDelete: idArray,
    routinesAdd: nodeRoutineListItemsAdd,
    routinesUpdate: nodeRoutineListItemsUpdate,
})

// Meant to be empty
const nodeStartAdd = yup.object().shape({
})
const nodeStartUpdate = yup.object().shape({
})

export const nodeAdd = yup.object().shape({
    description,
    title: title.required(),
    type: type.required(),
    nodeCombineAdd,
    nodeDecisionAdd,
    nodeEndAdd,
    nodeLoopAdd,
    nodeRedirectAdd,
    nodeRoutineListAdd,
    nodeStartAdd,
    nextId: id,
    previousId: id,
    routineId: id.required(),
})
/**
 * A node always contains one data type. Since this is known, there is no need 
 * to specify a disconnect or delete
 */
export const nodeUpdate = yup.object().shape({
    id,
    description,
    title,
    type,
    nodeCombineAdd,
    nodeCombineUpdate,
    nodeDecisionAdd,
    nodeDecisionUpdate,
    nodeEndAdd,
    nodeEndUpdate,
    nodeLoopAdd,
    nodeLoopUpdate,
    nodeRedirectAdd,
    nodeRedirectUpdate,
    nodeRoutineListAdd,
    nodeRoutineListUpdate,
    nodeStartAdd,
    nodeStartUpdate,
    nextId: id,
    previousId: id,
    routineId: id,
})

export const nodesAdd = yup.array().of(nodeAdd.required()).optional();
export const nodesUpdate = yup.array().of(nodeUpdate.required()).optional();