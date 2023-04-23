export const getLocalStorageKeys = ({ prefix = "", suffix = "", }) => {
    const keys = [];
    for (let i = 0; i < localStorage.length; i++) {
        const key = localStorage.key(i);
        if (key && key.startsWith(prefix) && key.endsWith(suffix)) {
            keys.push(key);
        }
    }
    return keys;
};
//# sourceMappingURL=localStorage.js.map