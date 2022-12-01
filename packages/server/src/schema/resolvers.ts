/**
 * Resolvers for GraphQL unions. 
 * The basic structure is: 
 *  1. Find one or more fields which are unique to an object type, and will probably be part of the query.
 *  2. If the object contains one of those fields, it must be of that type.
 *  3. Repeat for all other object types in the union
 */

import { logger } from "../events/logger";
import { GraphQLModelType } from "../models/types";

export const resolveCommentedOn = (obj: any): GraphQLModelType => {
    if (!obj) {
        logger.error('Null or undefined passed to resolveCommentedOn', { trace: '0217' });
        return 'Standard';
    }
    // Only a Standard has a type field
    if (obj.hasOwnProperty('type')) return 'Standard';
    // Only a routine has a complexity field
    if (obj.hasOwnProperty('complexity')) return 'Routine';
    return 'Project';
}

export const resolveNodeData = (obj: any): GraphQLModelType => {
    if (!obj) {
        logger.error('Null or undefined passed to resolveNodeData', { trace: '0218' });
        return 'NodeEnd';
    }
    // Only NodeEnd has wasSuccessful field
    if (obj.hasOwnProperty('wasSuccessful')) return 'NodeEnd';;
    return'NodeRoutineList';
}

export const resolveProjectOrRoutine = (obj: any): GraphQLModelType => {
    if (!obj) {
        logger.error('Null or undefined passed to resolveProjectOrRoutine', { trace: '0219' });
        return 'Project'
    }
    // Only a project has a handle field
    if (obj.hasOwnProperty('handle')) return 'Project';
    return 'Routine';
}

export const resolveProjectOrOrganization = (obj: any): GraphQLModelType => {
    if (!obj) {
        logger.error('Null or undefined passed to resolveProjectOrOrganization', { trace: '0220' });
        return 'Project'
    }
    // Only a project has a score field
    if (obj.hasOwnProperty('score')) return 'Project';
    return 'Organization';
}

export const resolveProjectOrOrganizationOrRoutineOrStandardOrUser = (obj: any): GraphQLModelType => {
    if (!obj) {
        logger.error('Null or undefined passed to resolveProjectOrOrganizationOrRoutineOrStandardOrUser', { trace: '0221' });
        return 'Routine'
    }
    // Only a routine has a complexity field
    if (obj.hasOwnProperty('complexity')) return 'Routine';
    // Only a user has an untranslated name field
    if (obj.hasOwnProperty('name')) return 'User';
    // Out of the remaining types, only an organization does not have isUpvoted field
    if (!obj.hasOwnProperty('isUpvoted')) return 'Organization';
    // Out of the remaining types, only a project has a handle field
    if (obj.hasOwnProperty('handle')) return 'Project';
    // There is only one remaining type, the standard
    return 'Standard';
}

export const resolveContributor = (obj: any): GraphQLModelType => {
    if (!obj) {
        logger.error('Null or undefined passed to resolveContributor', { trace: '0222' });
        return 'User'
    }
    // Only a user has a name field
    if (obj.hasOwnProperty('name')) return 'User';
    return 'Organization';
}

export const resolveStarTo = (obj: any): GraphQLModelType => {
    if (!obj) {
        logger.error('Null or undefined passed to resolveStarTo', { trace: '0223' });
        return 'Standard'
    }
    if (obj.hasOwnProperty('yup')) return 'Standard';
    if (obj.hasOwnProperty('complexity')) return 'Routine';
    if (obj.hasOwnProperty('isComplete')) return 'Project';
    if (obj.hasOwnProperty('isOpenToNewMembers')) return 'Organization';
    if (obj.hasOwnProperty('name')) return 'User';
    if (obj.hasOwnProperty('tag')) return 'Tag';
    return 'Comment';
}

export const resolveVoteTo = (obj: any): GraphQLModelType => {
    if (!obj) {
        logger.error('Null or undefined passed to resolveVoteTo', { trace: '0224' });
        return 'Standard'
    }
    if (obj.hasOwnProperty('type')) return 'Standard';
    if (obj.hasOwnProperty('isComplete')) return 'Project';
    return 'Routine';
}