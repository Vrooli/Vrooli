export const errorToCode = (error) => {
    if (error.graphQLErrors) {
        for (const err of error.graphQLErrors) {
            if (err.extensions?.code) {
                return err.extensions.code;
            }
        }
    }
    return "ErrorUnknown";
};
export const errorToMessage = (error) => {
    if (error.graphQLErrors) {
        for (const err of error.graphQLErrors) {
            if (err.message) {
                return err.message;
            }
        }
    }
    return errorToCode(error);
};
export const hasErrorCode = (error, code) => {
    return Array.isArray(error.graphQLErrors) && error.graphQLErrors.some(e => e.extensions?.code === code);
};
//# sourceMappingURL=errorParser.js.map