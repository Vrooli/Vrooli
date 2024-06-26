import { InputType } from "@local/shared";
import { useMemo } from "react";
import { GeneratedCheckbox } from "../GeneratedCheckbox/GeneratedCheckbox";
import { GeneratedCodeInput } from "../GeneratedCodeInput/GeneratedCodeInput";
import { GeneratedDropzone } from "../GeneratedDropzone/GeneratedDropzone";
import { GeneratedIntegerInput } from "../GeneratedIntegerInput/GeneratedIntegerInput";
import { GeneratedLanguageInput } from "../GeneratedLanguageInput/GeneratedLanguageInput";
import { GeneratedRadio } from "../GeneratedRadio/GeneratedRadio";
import { GeneratedSelector } from "../GeneratedSelector/GeneratedSelector";
import { GeneratedSlider } from "../GeneratedSlider/GeneratedSlider";
import { GeneratedSwitch } from "../GeneratedSwitch/GeneratedSwitch";
import { GeneratedTagSelector } from "../GeneratedTagSelector/GeneratedTagSelector";
import { GeneratedTextInput } from "../GeneratedTextInput/GeneratedTextInput";
import { GeneratedInputComponentProps } from "../types";

/**
 * Maps a data input type string to its corresponding component generator function
 */
const typeMap: { [key in InputType]: (props: GeneratedInputComponentProps) => JSX.Element } = {
    [InputType.Checkbox]: GeneratedCheckbox,
    [InputType.Dropzone]: GeneratedDropzone,
    [InputType.JSON]: GeneratedCodeInput,
    [InputType.IntegerInput]: GeneratedIntegerInput,
    [InputType.LanguageInput]: GeneratedLanguageInput,
    [InputType.Radio]: GeneratedRadio,
    [InputType.Selector]: GeneratedSelector,
    [InputType.Slider]: GeneratedSlider,
    [InputType.Switch]: GeneratedSwitch,
    [InputType.TagSelector]: GeneratedTagSelector,
    [InputType.Text]: GeneratedTextInput,
}

export const GeneratedInputComponent = (props: GeneratedInputComponentProps) => {
    console.log('rendering input component', props.fieldData.type);
    const InputComponent = useMemo(() => typeMap[props.fieldData.type], [props.fieldData.type]);
    return <InputComponent {...props} />
}