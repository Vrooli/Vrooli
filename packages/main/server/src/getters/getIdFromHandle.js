export async function getIdFromHandle({ handle, objectType, prisma, }) {
    const where = { handle };
    const select = { id: true };
    const query = { where, select };
    const id = objectType === "Organization" ? await prisma.organization.findFirst(query) :
        objectType === "Project" ? await prisma.project.findFirst(query) :
            await prisma.user.findFirst(query);
    return id?.id;
}
//# sourceMappingURL=getIdFromHandle.js.map