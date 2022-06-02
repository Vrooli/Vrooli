import { InputType } from "@local/shared";
import { FieldData } from "forms/types";
import { Standard } from "types";

/**
 * Converts Standard object to a FieldData shape. 
 * This can be used to generate input objects
 * @param standard Standard object
 * @param fieldName Name of the field
 * @returns FieldData shape
 */
export const standardToFieldData = (
    standard: {
        props: Standard['props'];
        name: Standard['name'];
        type: Standard['type'];
        yup: Standard['yup'];
    } | null | undefined,
    fieldName: string
): FieldData | null => {
    // Check props
    if (!standard) return null;
    // Props are stored as JSON, so they must be parsed
    let props: any;
    let yup: any | undefined = undefined;
    try {
        props = JSON.parse(standard.props);
        if (standard.yup) yup = JSON.parse(standard.yup);
    } catch (error) {
        console.error('Error parsing props/yup', error);
        return null;
    }
    return {
        fieldName,
        label: standard.name,
        type: standard.type as InputType,
        props,
        yup,
    }
}