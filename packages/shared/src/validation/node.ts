/**
 * Nodes are a bit more complicated than the other types because they come in
 * several variations. Nonetheless, they are handled in the same way - connect, 
 * disconnect, delete, create, and update
 */
import { description, idArray, id, title } from './base';
import * as yup from 'yup';
import { NodeType } from '../consts';

const condition = yup.string().max(2048).optional();
const isOptional = yup.boolean().optional();
const loops = yup.number().integer().min(0).max(100).optional();
const maxLoops = yup.number().integer().min(1).max(100).optional();
const type = yup.string().oneOf(Object.values(NodeType)).optional();
const wasSuccessful = yup.boolean().optional();

const nodeCombineCreate = yup.object().shape({
    from: idArray.required(),
})
const nodeCombineUpdate = yup.object().shape({
    id: id.required(),
    from: idArray,
})

const decisionWhenCreate = yup.array().of(yup.object().shape({
    condition: condition.required(),
}).required()).optional();
const decisionWhenUpdate = yup.array().of(yup.object().shape({
    id: id.required(),
    condition: condition.required(),
}).required()).optional();
const decisionsCreate = yup.array().of(yup.object().shape({
    description,
    title: title.required(),
    whenCreate: decisionWhenCreate,
    toId: id,
}).required()).optional();
const decisionsUpdate = yup.array().of(yup.object().shape({
    id: id.required(),
    description,
    title,
    toId: id,
    whenCreate: decisionWhenCreate,
    whenUpdate: decisionWhenUpdate,
    whenDelete: idArray,
}).required()).optional();
const nodeDecisionCreate = yup.object().shape({
    decisionsCreate: decisionsCreate.required(),
})
const nodeDecisionUpdate = yup.object().shape({
    id: id.required(),
    decisionsCreate,
    decisionsUpdate,
    decisionsDelete: idArray,
})

const nodeEndCreate = yup.object().shape({
    wasSuccessful,
})
const nodeEndUpdate = yup.object().shape({
    id: id.required(),
    wasSuccessful,
})

const loopWhenCreate = yup.array().of(yup.object().shape({
    condition: condition.required(),
}).required()).optional();
const loopWhenUpdate = yup.array().of(yup.object().shape({
    id: id.required(),
    condition: condition.required(),
}).required()).optional();
const whilesCreate = yup.array().of(yup.object().shape({
    description,
    title: title.required(),
    whenCreate: loopWhenCreate,
    toId: id,
}).required()).optional();
const whilesUpdate = yup.array().of(yup.object().shape({
    id: id.required(),
    description,
    title,
    toId: id,
    whenCreate: loopWhenCreate,
    whenUpdate: loopWhenUpdate,
    whenDelete: idArray,
}).required()).optional();
const nodeLoopCreate = yup.object().shape({
    loops,
    maxLoops,
    whiles: whilesCreate.required(),
})
const nodeLoopUpdate = yup.object().shape({
    id: id.required(),
    loops,
    maxLoops,
    whilesCreate,
    whilesUpdate,
    decisionsDelete: idArray,
})

const nodeRoutineListItemsCreate = yup.array().of(yup.object().shape({
    description,
    title,
    isOptional,
    // Cannot create a routine directly from a node, as this causes a cyclic dependency
    routineConnect: id,
}).required()).optional();
const nodeRoutineListItemsUpdate = yup.array().of(yup.object().shape({
    id: id.required(),
    description,
    isOptional,
    title,
    // Cannot create or update a routine directly from a node, as this causes a cyclic dependency
    routineConnect: id,
    routineDisconnect: id,
    routineDelete: id,
}).required()).optional();
const nodeRoutineListCreate = yup.object().shape({
    isOrdered: yup.boolean().optional(),
    isOptional: yup.boolean().optional(),
    routinesConnect: idArray,
    routinesCreate: nodeRoutineListItemsCreate,
})
const nodeRoutineListUpdate = yup.object().shape({
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
    description,
    title: title.required(),
    type: type.required(),
    nodeCombineCreate,
    nodeDecisionCreate,
    nodeEndCreate,
    nodeLoopCreate,
    nodeRoutineListCreate,
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
    nodeCombineCreate,
    nodeCombineUpdate,
    nodeDecisionCreate,
    nodeDecisionUpdate,
    nodeEndCreate,
    nodeEndUpdate,
    nodeLoopCreate,
    nodeLoopUpdate,
    nodeRoutineListCreate,
    nodeRoutineListUpdate,
    nextId: id,
    previousId: id,
    routineId: id,
})

export const nodesCreate = yup.array().of(nodeCreate.required()).optional();
export const nodesUpdate = yup.array().of(nodeUpdate.required()).optional();