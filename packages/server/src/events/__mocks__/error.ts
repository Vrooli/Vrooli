export class CustomError extends Error {
    constructor(traceBase: string, errorCode: string, _languages: string[], _data?: any) {
        super(errorCode);
        // This is a mock, so you might not want to replicate the full behavior.
        // Just the essentials for your tests to run. You can expand on this as needed.
        this.name = errorCode;
    }
}
