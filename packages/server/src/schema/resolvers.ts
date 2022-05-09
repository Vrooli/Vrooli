/**
 * Resolvers for GraphQL unions
 */

import { GraphQLModelType } from "../models";

export const resolveCommentedOn = (obj: any): GraphQLModelType => {
    // Only a Standard has an isFile field
    if (obj.hasOwnProperty('isFile')) return GraphQLModelType.Standard;
    // Only a Project has a name field
    if (obj.hasOwnProperty('isComplete')) return GraphQLModelType.Project;
    return GraphQLModelType.Routine;
}

export const resolveNodeData = (obj: any): GraphQLModelType => {
    // Only NodeEnd has wasSuccessful field
    if (obj.hasOwnProperty('wasSuccessful')) return GraphQLModelType.NodeEnd;
    return GraphQLModelType.NodeRoutineList;
}

export const resolveProjectOrRoutine = (obj: any): GraphQLModelType => {
    // Only a project has a handle field
    if (obj.hasOwnProperty('handle')) return GraphQLModelType.Project;
    return GraphQLModelType.Routine;
}

export const resolveProjectOrOrganization = (obj: any): GraphQLModelType => {
    // Only a project has a score field
    if (obj.hasOwnProperty('score')) return GraphQLModelType.Project;
    return GraphQLModelType.Organization;
}

export const resolveProjectOrOrganizationOrRoutineOrStandardOrUser = (obj: any): GraphQLModelType => {
    // Only a routine has a complexity field
    if (obj.hasOwnProperty('complexity')) return GraphQLModelType.Routine;
    // Only a user has an untranslated name field
    if (obj.hasOwnProperty('name')) return GraphQLModelType.User;
    // Out of the remaining types, only an organization does not have isUpvoted field
    if (!obj.hasOwnProperty('isUpvoted')) return GraphQLModelType.Organization;
    // Out of the remaining types, only a project has a handle field
    if (obj.hasOwnProperty('handle')) return GraphQLModelType.Project;
    // There is only one remaining type, the standard
    return GraphQLModelType.Standard;
}

export const resolveContributor = (obj: any): GraphQLModelType => {
    // Only a user has a name field
    if (obj.hasOwnProperty('name')) return GraphQLModelType.User;
    return GraphQLModelType.Organization;
}

export const resolveStarTo = (obj: any): GraphQLModelType => {
    console.log('RESOLVE STAR TO', JSON.stringify(obj));
    if (obj.hasOwnProperty('isFile')) return GraphQLModelType.Standard;
    if (obj.hasOwnProperty('complexity')) return GraphQLModelType.Routine;
    if (obj.hasOwnProperty('isComplete')) return GraphQLModelType.Project;
    if (obj.hasOwnProperty('isOpenToNewMembers')) return GraphQLModelType.Organization;
    if (obj.hasOwnProperty('name')) return GraphQLModelType.User;
    if (obj.hasOwnProperty('tag')) return GraphQLModelType.Tag;
    return GraphQLModelType.Comment;
}

export const resolveVoteTo = (obj: any): GraphQLModelType => {
    if (obj.hasOwnProperty('isFile')) return GraphQLModelType.Standard;
    if (obj.hasOwnProperty('isComplete')) return GraphQLModelType.Project;
    return GraphQLModelType.Routine;
}