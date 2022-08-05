import { InputType } from "@local/shared";
import { FieldData } from "forms/types";
import { Standard } from "types";

export interface StandardToFieldDataProps {
    description?: string | null;
    fieldName: string;
    name: Standard['name'];
    props: Standard['props'];
    type: Standard['type'];
    yup: Standard['yup'] | null | undefined;
}

/**
 * Converts Standard object to a FieldData shape. 
 * This can be used to generate input objects
 * @returns FieldData shape
 */
export const standardToFieldData = ({
    description,
    fieldName,
    name,
    props,
    type,
    yup,
}: StandardToFieldDataProps): FieldData | null => {
    // Props are stored as JSON, so they must be parsed
    let parsedProps: any;
    let parsedYup: any | undefined = undefined;
    try {
        parsedProps = JSON.parse(props);
        if (yup) yup = JSON.parse(yup);
    } catch (error) {
        console.error('Error parsing props/yup', error);
        return null;
    }
    return {
        description: description ?? undefined,
        fieldName,
        label: name,
        type: type as InputType,
        props: parsedProps,
        yup: parsedYup,
    }
}