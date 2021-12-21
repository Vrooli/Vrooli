// Helper functions for building Prisma queries

export const idArrayQuery = (ids: string[]): any => {
    if (!Array.isArray(ids) || ids.length === 0) return undefined;
    return ({ id: { in: ids}})
}