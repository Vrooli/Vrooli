export { Slider, SliderFactory } from "./Slider.js";
export type { SliderProps, SliderVariant, SliderSize, SliderMark } from "./Slider.js";

// Export slider utilities for advanced usage
export {
    SLIDER_CONFIG,
    SLIDER_COLORS,
    buildSliderContainerClasses,
    buildTrackClasses,
    buildFilledTrackClasses,
    buildThumbClasses,
    calculateSliderPosition,
    calculateValueFromPosition,
    getCustomSliderStyle,
} from "./sliderStyles.js";

// Re-export Switch components from their directory
export { Switch, SwitchFactory } from "./Switch/index.js";
export type { SwitchProps } from "./Switch/index.js";

// Export Radio components
export { Radio, PrimaryRadio, SecondaryRadio, DangerRadio } from "./Radio.js";
export type { RadioProps, RadioVariant, RadioSize } from "./types.js";

// Export Checkbox components
export { Checkbox, PrimaryCheckbox, SecondaryCheckbox, DangerCheckbox, SuccessCheckbox } from "./Checkbox.js";
export type { CheckboxProps, CheckboxVariant, CheckboxSize } from "./types.js";

// Export FormControlLabel component
export { FormControlLabel } from "./FormControlLabel.js";
export type { FormControlLabelProps, FormControlLabelPlacement } from "./types.js";

// Export FormGroup component
export { FormGroup } from "./FormGroup.js";
export type { FormGroupProps } from "./types.js";