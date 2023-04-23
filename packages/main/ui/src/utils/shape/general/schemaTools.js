export const standardVersionToFieldData = ({ description, fieldName, helpText, name, props, standardType, yup, }) => {
    console.log("standardversiontofielddata", fieldName, description, helpText);
    let parsedProps;
    const parsedYup = undefined;
    try {
        parsedProps = JSON.parse(props);
        if (yup)
            yup = JSON.parse(yup);
    }
    catch (error) {
        console.error("Error parsing props/yup", error);
        return null;
    }
    return {
        description,
        fieldName,
        helpText,
        label: name,
        type: standardType,
        props: parsedProps,
        yup: parsedYup,
    };
};
//# sourceMappingURL=schemaTools.js.map