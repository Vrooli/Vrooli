import { jsx as _jsx } from "react/jsx-runtime";
import { InputType } from "@local/consts";
import { useMemo } from "react";
import { GeneratedCheckbox } from "../GeneratedCheckbox/GeneratedCheckbox";
import { GeneratedDropzone } from "../GeneratedDropzone/GeneratedDropzone";
import { GeneratedIntegerInput } from "../GeneratedIntegerInput/GeneratedIntegerInput";
import { GeneratedJsonInput } from "../GeneratedJsonInput/GeneratedJsonInput";
import { GeneratedLanguageInput } from "../GeneratedLanguageInput/GeneratedLanguageInput";
import { GeneratedMarkdownInput } from "../GeneratedMarkdownInput/GeneratedMarkdownInput";
import { GeneratedRadio } from "../GeneratedRadio/GeneratedRadio";
import { GeneratedSelector } from "../GeneratedSelector/GeneratedSelector";
import { GeneratedSlider } from "../GeneratedSlider/GeneratedSlider";
import { GeneratedSwitch } from "../GeneratedSwitch/GeneratedSwitch";
import { GeneratedTagSelector } from "../GeneratedTagSelector/GeneratedTagSelector";
import { GeneratedTextField } from "../GeneratedTextField/GeneratedTextField";
const typeMap = {
    [InputType.Checkbox]: GeneratedCheckbox,
    [InputType.Dropzone]: GeneratedDropzone,
    [InputType.JSON]: GeneratedJsonInput,
    [InputType.IntegerInput]: GeneratedIntegerInput,
    [InputType.LanguageInput]: GeneratedLanguageInput,
    [InputType.Markdown]: GeneratedMarkdownInput,
    [InputType.Prompt]: GeneratedTextField,
    [InputType.Radio]: GeneratedRadio,
    [InputType.Selector]: GeneratedSelector,
    [InputType.Slider]: GeneratedSlider,
    [InputType.Switch]: GeneratedSwitch,
    [InputType.TagSelector]: GeneratedTagSelector,
    [InputType.TextField]: GeneratedTextField,
};
export const GeneratedInputComponent = (props) => {
    console.log("rendering input component", props.fieldData.type);
    const InputComponent = useMemo(() => typeMap[props.fieldData.type], [props.fieldData.type]);
    return _jsx(InputComponent, { ...props });
};
//# sourceMappingURL=GeneratedInputComponent.js.map