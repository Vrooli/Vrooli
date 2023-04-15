import { StandardVersion } from "@shared/consts";
import { FieldData } from "forms/types";

export interface StandardVersionToFieldDataProps {
    description?: StandardVersion['translations'][0]['description'];
    fieldName: string;
    helpText: string | null | undefined;
    name: StandardVersion['translations'][0]['name'];
    props: StandardVersion['props'];
    standardType: StandardVersion['standardType'];
    yup: StandardVersion['yup'] | null | undefined;
}

/**
 * Converts StandardVersion object to a FieldData shape. 
 * This can be used to generate input objects
 * @returns FieldData shape
 */
export const standardVersionToFieldData = ({
    description,
    fieldName,
    helpText,
    name,
    props,
    standardType,
    yup,
}: StandardVersionToFieldDataProps): FieldData | null => {
    console.log('standardversiontofielddata', fieldName, description, helpText)
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
        description,
        fieldName,
        helpText,
        label: name,
        type: standardType as any,
        props: parsedProps,
        yup: parsedYup,
    }
}