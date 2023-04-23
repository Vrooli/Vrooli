import { jsx as _jsx } from "react/jsx-runtime";
import { useMemo } from "react";
import { Selector } from "../../Selector/Selector";
export const GeneratedSelector = ({ disabled, fieldData, index, }) => {
    console.log("rendering selector");
    const props = useMemo(() => fieldData.props, [fieldData.props]);
    return (_jsx(Selector, { autoFocus: index === 0, disabled: disabled, options: props.options, getOptionLabel: props.getOptionLabel, name: fieldData.fieldName, fullWidth: true, inputAriaLabel: `select-input-${fieldData.fieldName}`, noneOption: props.noneOption, label: fieldData.label, tabIndex: index }, `field-${fieldData.fieldName}-${index}`));
};
//# sourceMappingURL=GeneratedSelector.js.map