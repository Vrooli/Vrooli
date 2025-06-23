export { Switch, SwitchFactory } from "./Switch.js";
export type { SwitchProps, SwitchVariant, SwitchSize, LabelPosition } from "./types.js";

// Export switch utilities for advanced usage
export {
    SWITCH_CONFIG,
    SWITCH_COLORS,
    SWITCH_TRACK_STYLES,
    SWITCH_THUMB_STYLES,
    SWITCH_LABEL_STYLES,
    BASE_SWITCH_STYLES,
    buildSwitchClasses,
    getTrackDimensions,
    getThumbDimensions,
    getThumbPosition,
    getCustomSwitchStyle,
    getFocusRingColor,
} from "./switchStyles.js";
