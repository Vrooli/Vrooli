import { minVersionTest } from './minVersionTest';

describe('minVersionTest function tests', () => {
    const minVersion = '1.0.0';

    test('version meets the minimum version requirement', () => {
        const [, , testFn] = minVersionTest(minVersion);
        expect(testFn('1.0.1')).toBe(true);
    });

    test('version does not meet the minimum version requirement', () => {
        const [, , testFn] = minVersionTest(minVersion);
        expect(testFn('0.9.9')).toBe(false);
    });

    test('undefined version', () => {
        const [, , testFn] = minVersionTest(minVersion);
        expect(testFn(undefined)).toBe(true);
    });

    test('minimum version as input version', () => {
        const [, , testFn] = minVersionTest(minVersion);
        expect(testFn(minVersion)).toBe(true);
    });
});
