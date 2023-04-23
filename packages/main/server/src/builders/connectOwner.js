export const connectOwner = (createInput, session) => {
    if (createInput.organizationConnect) {
        return ({ ownedByOrganization: { connect: { id: createInput.organizationConnect } } });
    }
    if (createInput.userConnect) {
        return ({ ownedByUser: { connect: { id: createInput.userConnect } } });
    }
    return ({ ownedByUser: { connect: { id: session.id } } });
};
//# sourceMappingURL=connectOwner.js.map