import { type SessionUser } from "@vrooli/shared";

type ConnectOwnerInput = {
    userConnect?: string | null | undefined;
    teamConnect?: string | null | undefined;
}

/**
 * Partial query to connect an owner to a new object
 * @param createInput Input to create the new object
 * @param session Current user
 * @returns Partial query to connect an owner to a new object
 */
export const connectOwner = <T extends ConnectOwnerInput>(createInput: T, session: SessionUser) => {
    // If team is specified (including empty string), connect to that
    if (createInput.teamConnect !== null && createInput.teamConnect !== undefined) {
        return ({ ownedByTeam: { connect: { id: createInput.teamConnect } } });
    }
    // If user is specified (including empty string), connect to that
    if (createInput.userConnect !== null && createInput.userConnect !== undefined) {
        return ({ ownedByUser: { connect: { id: createInput.userConnect } } });
    }
    // If neither is specified, connect to the current user
    return ({ ownedByUser: { connect: { id: session.id } } });
};
