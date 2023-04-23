import { valueFromDot } from "./objectTools";
export const addToArray = (array, value) => {
    return [...array, value];
};
export const updateArray = (array, index, value) => {
    if (JSON.stringify(array[index]) === JSON.stringify(value) || index < 0 || index >= array.length)
        return array;
    const copy = [...array];
    copy[index] = value;
    return copy;
};
export const deleteArrayIndex = (array, index) => {
    return array.filter((_, i) => i !== index);
};
export const deleteArrayObject = (array, obj) => {
    const index = array.findIndex(obj);
    if (index !== -1) {
        const copy = [...array];
        copy.splice(index, 1);
        return copy;
    }
};
export const findWithAttr = (array, attr, value) => {
    for (let i = 0; i < array.length; i += 1) {
        if (array[i][attr] === value) {
            return i;
        }
    }
    return -1;
};
export const moveArrayIndex = (array, from, to) => {
    const copy = [...array];
    copy.splice(to, 0, copy.splice(from, 1)[0]);
    return copy;
};
export const rotateArray = (array, to_right = true) => {
    if (array.length === 0)
        return array;
    const copy = [...array];
    if (to_right) {
        const last_elem = copy.pop();
        copy.unshift(last_elem);
        return copy;
    }
    else {
        const first_elem = copy.shift();
        copy.push(first_elem);
        return copy;
    }
};
export function mapIfExists(object, notation, operation) {
    const value = valueFromDot(object, notation);
    if (!Array.isArray(value))
        return null;
    return value.map(v => operation(v));
}
//# sourceMappingURL=arrayTools.js.map