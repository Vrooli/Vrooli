/**
 * Nodes are a bit more complicated than the other types because they come in
 * several variations. Nonetheless, they are handled in the same way - connect, 
 * disconnect, delete, add, and update
 */
import { description, idArray, id, title } from './';
import * as yup from 'yup';
import { NodeType } from '../consts';

const condition = yup.string().max(2048).optional();
const type = yup.string().oneOf(Object.values(NodeType)).optional();
const wasSuccessful = yup.boolean().optional();

const dataCombineAdd = yup.object().shape({
    from: idArray.required(),
    toId: id,
})
const dataCombineUpdate = yup.object().shape({
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
const dataDecisionAdd = yup.object().shape({
    decisionsAdd: decisionsAdd.required(),
})
const dataDecisionUpdate = yup.object().shape({
    id: id.required(),
    decisionsAdd,
    decisionsUpdate,
    decisionsDelete: idArray,
})

const dataEndAdd = yup.object().shape({
    wasSuccessful,
})
const dataEndUpdate = yup.object().shape({
    id: id.required(),
    wasSuccessful,
})

// TODO define
const dataLoopAdd = yup.object().shape({
    id,
})
// TODO define
const dataLoopUpdate = yup.object().shape({
    id: id.required(),
})

// TODO define
const dataRedirectAdd = yup.object().shape({
    id,
})
// TODO define
const dataRedirectUpdate = yup.object().shape({
    id: id.required(),
})

const dataRoutineListItemsAdd = yup.array().of(yup.object().shape({
    description,
    title,
    // Cannot add a routine directly from a node, as this causes a cyclic dependency
    routineConnect: id,
}).required()).optional();
const dataRoutineListItemsUpdate = yup.array().of(yup.object().shape({
    id: id.required(),
    description,
    title,
    // Cannot add or update a routine directly from a node, as this causes a cyclic dependency
    routineConnect: id,
    routineDisconnect: id,
    routineDelete: id,
}).required()).optional();
const dataRoutineListAdd = yup.object().shape({
    isOrdered: yup.boolean().optional(),
    isOptional: yup.boolean().optional(),
    routinesConnect: idArray,
    routinesAdd: dataRoutineListItemsAdd,
})
const dataRoutineListUpdate = yup.object().shape({
    id: id.required(),
    isOrdered: yup.boolean().optional(),
    isOptional: yup.boolean().optional(),
    routinesConnect: idArray,
    routinesDisconnect: idArray,
    routinesDelete: idArray,
    routinesAdd: dataRoutineListItemsAdd,
    routinesUpdate: dataRoutineListItemsUpdate,
})

// Meant to be empty
const dataStartAdd = yup.object().shape({
})
const dataStartUpdate = yup.object().shape({
})

export const nodeAdd = yup.object().shape({
    id,
    description,
    title: title.required(),
    type: type.required(),
    dataCombineAdd,
    dataDecisionAdd,
    dataEndAdd,
    dataLoopAdd,
    dataRedirectAdd,
    dataRoutineListAdd,
    dataStartAdd,
    nextId: id,
    previousId: id,
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
    dataCombineAdd,
    dataCombineUpdate,
    dataDecisionAdd,
    dataDecisionUpdate,
    dataEndAdd,
    dataEndUpdate,
    dataLoopAdd,
    dataLoopUpdate,
    dataRedirectAdd,
    dataRedirectUpdate,
    dataRoutineListAdd,
    dataRoutineListUpdate,
    dataStartAdd,
    dataStartUpdate,
    nextId: id,
    previousId: id,
})

export const nodesAdd = yup.array().of(nodeAdd.required()).optional();
export const nodesUpdate = yup.array().of(nodeUpdate.required()).optional();