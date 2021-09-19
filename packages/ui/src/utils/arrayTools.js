// Functions for manipulating arrays, especially state arrays

import { valueFromDot } from "./objectTools";

export const addToArray = (array, value) => {
    return [...array, value];
}

export const updateArray = (array, index, value) => {
    if (JSON.stringify(array[index]) === JSON.stringify(value)) return array;
    let copy = [...array];
    copy[index] = value;
    return copy;
}

export const deleteArrayIndex = (array, index) => {
    return array.filter((_, i) => i !== index);
}

export const deleteArrayObject = (array, obj) => {
    var index = array.findIndex(obj);
    if (index !== -1) {
        let copy = [...array];
        copy.splice(index, 1);
        return copy;
    }
}

export const findWithAttr = (array, attr, value) => {
    for(let i = 0; i < array.length; i += 1) {
        if(array[i][attr] === value) {
            return i;
        }
    }
    return -1;
}

export const moveArrayIndex = (array, from, to) => {
    let copy = [...array];
    copy.splice(to, 0, copy.splice(from, 1)[0]);
    return copy;
}

// Shifts everything to the right, and puts the last element in the beginning
export const rotateArray = (array, to_right = true) => {
    if (array.length === 0) return array;
    let copy = [...array];
    if (to_right) {
        let last_elem = copy.pop();
        copy.unshift(last_elem);
        return copy;
    } else {
        let first_elem = copy.shift();
        copy.push(first_elem);
        return copy;
    }
}

// If dot notation key exists in object, perform operation and return the results
export function mapIfExists(object, notation, operation) {
    const value = valueFromDot(object, notation);
    if (!Array.isArray(value)) return null;
    return value.map(v => operation(v))
}