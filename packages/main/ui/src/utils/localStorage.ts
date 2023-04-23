interface GetLocalStorageKeysProps {
    prefix?: string;
    suffix?: string;
}

/**
 * Find all keys in localStorage matching the specified options
 * @returns Array of keys
 */
export const getLocalStorageKeys = ({
    prefix = "",
    suffix = "",
}: GetLocalStorageKeysProps): string[] => {
    const keys: string[] = [];
    for (let i = 0; i < localStorage.length; i++) {
        const key = localStorage.key(i);
        if (key && key.startsWith(prefix) && key.endsWith(suffix)) {
            keys.push(key);
        }
    }
    return keys;
};
