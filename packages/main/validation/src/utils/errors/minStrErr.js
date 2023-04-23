export const minStrErr = (params) => {
    const amountUnder = params.min - params.value.length;
    if (amountUnder === 1) {
        return "1 character under the limit";
    }
    else {
        return `${amountUnder} characters under the limit`;
    }
};
//# sourceMappingURL=minStrErr.js.map