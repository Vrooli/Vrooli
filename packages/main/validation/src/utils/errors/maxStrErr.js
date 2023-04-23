export const maxStrErr = (params) => {
    const amountOver = params.value.length - params.max;
    if (amountOver === 1) {
        return "1 character over the limit";
    }
    else {
        return `${amountOver} characters over the limit`;
    }
};
//# sourceMappingURL=maxStrErr.js.map