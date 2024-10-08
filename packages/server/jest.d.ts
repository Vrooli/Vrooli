declare namespace jest {
    interface Matchers<R> {
        toBeBigInt(expected: bigint | string | number): R;
    }
}