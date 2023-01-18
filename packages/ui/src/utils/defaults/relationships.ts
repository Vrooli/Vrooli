import { Session } from "@shared/consts";
import { userFromSession } from "components";
import { RelationshipsObject } from "components/inputs/types";

/**
 * Creates default relationships object, which is used to store values from RelationshipButtons. 
 * @param isEditing Whether or not this is being used for editing an object. If true, we set the owner to the current user.
 * @param session The current user's session. Only needed if this is being used for creating/updating an object, 
 * as we use the data to set the owner.
 * @returns A default relationships object.
 */
export const defaultRelationships = <
    IsEditing extends boolean = false
>(isEditing: IsEditing, session: IsEditing extends true ? Session : null): RelationshipsObject => ({
    isComplete: false,
    isPrivate: false,
    owner: isEditing ? userFromSession(session as Session) : null,
    parent: null,
    project: null,
})