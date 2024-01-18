import { getMinVersion } from "./getMinVersion";

describe('getMinVersion function tests', () => {
    test('versions in random order', () => {
        const versions = ['1.0.0', '2.0.0', '1.5.0'];
        const result = getMinVersion(versions);
        expect(result).toBe('2.0.0');
    });

    test('versions in ascending order', () => {
        const versions = ['1.0.0', '1.5.0', '2.0.0'];
        const result = getMinVersion(versions);
        expect(result).toBe('2.0.0');
    });

    test('versions in descending order', () => {
        const versions = ['2.0.0', '1.5.0', '1.0.0'];
        const result = getMinVersion(versions);
        expect(result).toBe('2.0.0');
    });

    test('empty list of versions', () => {
        const versions = [];
        const result = getMinVersion(versions);
        expect(result).toBe('0.0.1');
    });

    test('list with only one version', () => {
        const versions = ['1.0.0'];
        const result = getMinVersion(versions);
        expect(result).toBe('1.0.0');
    });

    test('list with duplicate versions', () => {
        const versions = ['1.0.0', '1.0.0', '2.0.0'];
        const result = getMinVersion(versions);
        expect(result).toBe('2.0.0');
    });
});
