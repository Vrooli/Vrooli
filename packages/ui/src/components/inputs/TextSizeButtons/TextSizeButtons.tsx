import { IconButton, Slider, SliderThumb, useTheme } from "@mui/material";
import { useThrottle } from "hooks/useThrottle";
import { AddIcon, CaseSensitiveIcon, MinusIcon } from "icons";
import { useCallback, useState } from "react";
import { FONT_SIZE_MAX, FONT_SIZE_MIN } from "utils/consts";
import { getCookie } from "utils/cookies";
import { PubSub } from "utils/pubsub";

const THROTTLE_MS = 50;

function ThumbComponent(props: React.HTMLAttributes<unknown>) {
    const { children, ...other } = props;
    return (
        <SliderThumb {...other}>
            {children}
            <CaseSensitiveIcon width="20px" height="20px" />
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

    const handleSliderChangeThrottled = useThrottle<[Event, number | number[]]>((event, newValue) => {
        handleSliderChange(Array.isArray(newValue) ?
            newValue.length > 0 ?
                newValue[0] :
                size : // If array was empty (shouldn't ever occur), fallback to the current size
            newValue);
    }, THROTTLE_MS);

    return (
        <div style={{ display: "flex", alignItems: "center" }}>
            <IconButton onClick={() => handleSliderChange(size - 1)} disabled={size === FONT_SIZE_MIN}>
                <MinusIcon fill={palette.secondary.main} />
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
            <IconButton onClick={() => handleSliderChange(size + 1)} disabled={size === FONT_SIZE_MAX}>
                <AddIcon fill={palette.secondary.main} />
            </IconButton>
        </div>
    );
}
