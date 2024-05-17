import { decodeValue, encodeValue } from './url';

describe('encodeValue and decodeValue', () => {
    const testCases = [
        {
            description: 'handles plain strings without percent signs',
            input: 'simple string',
            encoded: 'simple string',
            decoded: 'simple string'
        },
        {
            description: 'encodes and decodes percent signs in strings',
            input: '100% sure',
            encoded: '100%25 sure',
            decoded: '100% sure'
        },
        {
            description: 'recursively encodes and decodes objects',
            input: { key: '50% discount', nested: { prop: '20% off' } },
            encoded: { key: '50%25 discount', nested: { prop: '20%25 off' } },
            decoded: { key: '50% discount', nested: { prop: '20% off' } }
        },
        {
            description: 'recursively encodes and decodes arrays',
            input: ['10% milk', '5% fat'],
            encoded: ['10%25 milk', '5%25 fat'],
            decoded: ['10% milk', '5% fat']
        },
        {
            description: 'handles mixed types with arrays and objects',
            input: { percentages: ['30% rate', '40% area'], detail: { info: '60% done' } },
            encoded: { percentages: ['30%25 rate', '40%25 area'], detail: { info: '60%25 done' } },
            decoded: { percentages: ['30% rate', '40% area'], detail: { info: '60% done' } }
        },
        {
            description: 'does not modify non-string types',
            input: [10, 20.5, true, null, undefined],
            encoded: [10, 20.5, true, null, undefined],
            decoded: [10, 20.5, true, null, undefined]
        }
    ];

    testCases.forEach(({ description, input, encoded, decoded }) => {
        test(description, () => {
            const encodedValue = encodeValue(input);
            expect(encodedValue).toEqual(encoded);

            const decodedValue = decodeValue(encodedValue);
            expect(decodedValue).toEqual(decoded);
        });
    });
});
