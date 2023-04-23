import { jsx as _jsx } from "react/jsx-runtime";
import { useMemo } from "react";
import { IntegerInput } from "../../IntegerInput/IntegerInput";
export const GeneratedIntegerInput = ({ fieldData, index, }) => {
    console.log("rendering integer input");
    const props = useMemo(() => fieldData.props, [fieldData.props]);
    return (_jsx(IntegerInput, { autoFocus: index === 0, tabIndex: index, label: fieldData.label, min: props.min ?? 0, name: fieldData.fieldName, tooltip: props.tooltip ?? "" }, `field-${fieldData.fieldName}-${index}`));
};
//# sourceMappingURL=GeneratedIntegerInput.js.map