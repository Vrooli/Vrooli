/**
 * Calculates major, moderate, and minor versions from a version string
 * Ex: 1.2.3 => major = 1, moderate = 2, minor = 3
 * Ex: 1 => major = 1, moderate = 0, minor = 0
 * Ex: 1.2 => major = 1, moderate = 2, minor = 0
 * Ex: asdfasdf (or any other invalid number) => major = 1, moderate = 0, minor = 0
 * @param version Version string
 * @returns Major, moderate, and minor versions
 */
export const calculateVersionsFromString = (version: string): { major: number, moderate: number, minor: number } => {
    const [major, moderate, minor] = version.split('.').map(v => parseInt(v));
    return {
        major: major || 0,
        moderate: moderate || 0,
        minor: minor || 0
    }
}