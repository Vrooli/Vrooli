/** A weak string hash for things like cache keys. */
export const weakHash = (str: string) => {
    let hash = 0;
    if (str.length === 0) {
        return hash.toString();
    }
    for (let i = 0; i < str.length; i++) {
        const char = str.charCodeAt(i);
        hash = (hash << 5) - hash + char;
        hash = hash & hash; // Convert to 32bit integer
    }
    return hash.toString();
};

export const chatMatchHash = (userIds: string[], task?: string): string => {
    // Sort for consistent ordering
    let toHash = userIds.sort().join("|");
    // Append the task if it exists
    if (task) { toHash += `|${task}`; }
    // Create a  hash of the sortedIDs
    return weakHash(toHash);
};
