/**
 * Checks if the given value is a not null and not undefined.
 * @param value The value to check.
 * @returns True if the value is not null and not undefined, false otherwise.
 */
export function exists<T>(value: T | null | undefined): value is T {
    return value !== null && value !== undefined;
}
