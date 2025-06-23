import { IconButton } from "../../buttons/IconButton.js";
import Slider, { SliderThumb } from "@mui/material/Slider";
import { useTheme } from "@mui/material";
import { useCallback, useState } from "react";
import { useThrottle } from "../../../hooks/useThrottle.js";
import { IconCommon, IconText } from "../../../icons/Icons.js";
import { FONT_SIZE_MAX, FONT_SIZE_MIN } from "../../../utils/consts.js";
import { getCookie } from "../../../utils/localStorage.js";
import { PubSub } from "../../../utils/pubsub.js";

const THROTTLE_MS = 50;

function ThumbComponent(props: React.HTMLAttributes<unknown>) {
    const { children, ...other } = props;
    return (
        <SliderThumb
            aria-label="Text size"
            {...other}
        >
            {children}
            <IconText
                decorative
                name="CaseSensitive"
                size={20}
            />
        </SliderThumb>
    );
}

/**
 * Updates the font size of the entire app
 */
export function TextSizeButtons() {
    const { palette } = useTheme();

    const [size, setSize] = useState<number>(getCookie("FontSize"));

    const handleSliderChange = useCallback(function handleSliderChangeCallback(newValue: number) {
        if (newValue >= FONT_SIZE_MIN && newValue <= FONT_SIZE_MAX) {
            setSize(newValue);
            PubSub.get().publish("fontSize", newValue);
        }
    }, []);
    function decreaseByOne() {
        handleSliderChange(size - 1);
    }
    function increaseByOne() {
        handleSliderChange(size + 1);
    }

    const handleSliderChangeThrottled = useThrottle<[Event, number | number[]]>((event, newValue) => {
        handleSliderChange(Array.isArray(newValue) ?
            newValue.length > 0 ?
                newValue[0] :
                size : // If array was empty (shouldn't ever occur), fallback to the current size
            newValue);
    }, THROTTLE_MS);

    return (
        <div style={{ display: "flex", alignItems: "center" }}>
            <IconButton
                variant="transparent"
                size="md"
                aria-label="Decrease text size"
                disabled={size === FONT_SIZE_MIN}
                onClick={decreaseByOne}
            >
                <IconCommon
                    decorative
                    fill={palette.secondary.main}
                    name="Minus"
                />
            </IconButton>
            <Slider
                value={size}
                onChange={handleSliderChangeThrottled}
                valueLabelDisplay="auto"
                min={FONT_SIZE_MIN}
                max={FONT_SIZE_MAX}
                slots={{ thumb: ThumbComponent }}
                sx={{
                    color: palette.secondary.main,
                    flex: 1,
                    "& .MuiSlider-thumb": {
                        backgroundColor: palette.secondary.main,
                        color: palette.secondary.contrastText,
                        height: 30,
                        width: 30,
                        "&:hover, &.Mui-focusVisible": {
                            boxShadow: "initial",
                        },
                    },
                }}
            />
            <IconButton
                variant="transparent"
                size="md"
                aria-label="Increase text size"
                disabled={size === FONT_SIZE_MAX}
                onClick={increaseByOne}
            >
                <IconCommon
                    decorative
                    fill={palette.secondary.main}
                    name="Plus"
                />
            </IconButton>
        </div>
    );
}
