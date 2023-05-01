import { SessionUser } from "@local/shared";

type ConnectOwnerInput = {
    userConnect?: string | null | undefined;
    organizationConnect?: string | null | undefined;
}

/**
 * Partial query to connect an owner to a new object
 * @param createInput Input to create the new object
 * @param session Current user
 * @returns Partial query to connect an owner to a new object
 */
export const connectOwner = <T extends ConnectOwnerInput>(createInput: T, session: SessionUser) => {
    // If organization is specified, connect to that
    if (createInput.organizationConnect) {
        return ({ ownedByOrganization: { connect: { id: createInput.organizationConnect } } });
    }
    // If user is specified, connect to that
    if (createInput.userConnect) {
        return ({ ownedByUser: { connect: { id: createInput.userConnect } } });
    }
    // If neither is specified, connect to the current user
    return ({ ownedByUser: { connect: { id: session.id } } });
};
