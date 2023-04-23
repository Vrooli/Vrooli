import { jsx as _jsx } from "react/jsx-runtime";
import { useMemo } from "react";
import { MarkdownInput } from "../../MarkdownInput/MarkdownInput";
export const GeneratedMarkdownInput = ({ disabled, fieldData, index, }) => {
    console.log("rendering markdown input");
    const props = useMemo(() => fieldData.props, [fieldData.props]);
    return (_jsx(MarkdownInput, { disabled: disabled, name: fieldData.fieldName, placeholder: props.placeholder ?? fieldData.label, minRows: props.minRows }));
};
//# sourceMappingURL=GeneratedMarkdownInput.js.map