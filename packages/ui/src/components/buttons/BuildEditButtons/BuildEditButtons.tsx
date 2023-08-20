import { Box, Grid, Slider } from "@mui/material";
import { useThrottle } from "hooks/useThrottle";
import { useCallback } from "react";
import { GridSubmitButtons } from "../GridSubmitButtons/GridSubmitButtons";
import { BuildEditButtonsProps } from "../types";

export const BuildEditButtons = ({
    canSubmitMutate,
    canCancelMutate,
    errors,
    handleCancel,
    handleScaleChange,
    handleSubmit,
    isAdding,
    isEditing,
    loading,
    scale,
    zIndex,
}: BuildEditButtonsProps) => {

    const handleSliderChangeThrottled = useThrottle((delta) => {
        handleScaleChange(delta);
    }, 50);

    const handleSliderChange = useCallback((_event: Event, newValue: number | number[]) => {
        const delta = (newValue as number) - scale;
        console.log("handleSliderChange", delta);
        handleSliderChangeThrottled(delta);
    }, [scale, handleSliderChangeThrottled]);

    if (!isEditing) return null;
    return (
        <Box sx={{
            alignItems: "center",
            background: "transparent",
            display: "flex",
            justifyContent: "center",
            position: "absolute",
            zIndex: 2,
            bottom: 0,
            right: 0,
            paddingBottom: "calc(16px + env(safe-area-inset-bottom))",
            paddingLeft: "calc(16px + env(safe-area-inset-left))",
            paddingRight: "calc(16px + env(safe-area-inset-right))",
            height: "calc(64px + env(safe-area-inset-bottom))",
            width: "min(100%, 800px)",
        }}>
            <Box sx={{ width: "100%" }}>
                <Slider
                    aria-labelledby="range-slider"
                    disabled={!isEditing}
                    value={scale}
                    onChange={handleSliderChange}
                    valueLabelDisplay="off"
                    min={-3}
                    max={0.5}
                    step={0.1}
                    sx={{
                        marginBottom: 2,
                        "& .MuiSlider-root": {
                            width: "100%",
                        },
                    }}
                />
            </Box>
            <Grid container spacing={1} ml={2}>
                <GridSubmitButtons
                    disabledCancel={loading || !canCancelMutate}
                    disabledSubmit={loading || !canSubmitMutate}
                    display="page"
                    errors={errors}
                    hideTextOnMobile
                    isCreate={isAdding}
                    onCancel={handleCancel}
                    onSubmit={handleSubmit}
                    zIndex={zIndex}
                />
            </Grid>
        </Box>
    );
};
