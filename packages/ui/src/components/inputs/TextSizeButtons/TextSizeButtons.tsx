import { IconButton, Slider, SliderThumb, useTheme } from "@mui/material";
import { useThrottle } from "hooks/useThrottle";
import { AddIcon, CaseSensitiveIcon, MinusIcon } from "icons";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { getCookieFontSize } from "utils/cookies";
import { PubSub } from "utils/pubsub";

const smallestFontSize = 10;
const largestFontSize = 20;

const ThumbComponent = (props: React.HTMLAttributes<unknown>) => {
    const { children, ...other } = props;
    return (
        <SliderThumb {...other}>
            {children}
            <CaseSensitiveIcon width="20px" height="20px" />
        </SliderThumb>
    );
};

/**
 * Updates the font size of the entire app
 */
export const TextSizeButtons = () => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [size, setSize] = useState<number>(getCookieFontSize(14));

    const handleSliderChange = (newValue: number) => {
        if (newValue >= smallestFontSize && newValue <= largestFontSize) {
            setSize(newValue);
            PubSub.get().publishFontSize(newValue);
        }
    };

    const handleSliderChangeThrottled = useThrottle((event: Event, newValue: number | number[]) => {
        handleSliderChange(newValue as number);
    }, 50);

    return (
        <div style={{ display: "flex", alignItems: "center" }}>
            <IconButton onClick={() => handleSliderChange(size - 1)} disabled={size === smallestFontSize}>
                <MinusIcon fill={palette.secondary.main} />
            </IconButton>
            <Slider
                value={size}
                onChange={handleSliderChangeThrottled}
                valueLabelDisplay="auto"
                min={smallestFontSize}
                max={largestFontSize}
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
            <IconButton onClick={() => handleSliderChange(size + 1)} disabled={size === largestFontSize}>
                <AddIcon fill={palette.secondary.main} />
            </IconButton>
        </div>
    );
};
