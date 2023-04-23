import { jsx as _jsx } from "react/jsx-runtime";
import { useMemo } from "react";
import { JsonInput } from "../../JsonInput/JsonInput";
export const GeneratedJsonInput = ({ disabled, fieldData, index, }) => {
    console.log("rendering json input");
    const props = useMemo(() => fieldData.props, [fieldData.props]);
    return (_jsx(JsonInput, { disabled: disabled, format: props.format, variables: props.variables, placeholder: props.placeholder ?? fieldData.label, minRows: props.minRows, name: fieldData.fieldName }));
};
//# sourceMappingURL=GeneratedJsonInput.js.map