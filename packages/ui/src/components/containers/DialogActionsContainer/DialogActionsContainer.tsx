import { Box, Button, Grid, useTheme } from "@mui/material";
import { DialogActionsContainerProps, DialogActionItem } from "../types";
import { useCallback, useMemo } from "react";
import Measure from "react-measure";

export const DialogActionsContainer = ({
    fixed = true,
    actions,
    onResize,
}: DialogActionsContainerProps) => {
    const { palette } = useTheme();

    const handleResize = useCallback(({ bounds }: any) => {
        if (onResize) onResize({ width: bounds.width, height: bounds.height });
    }, [onResize]);

    /**
     * Convert actions into grid items
     */
    const gridItems: JSX.Element[] = useMemo(() => {
        // Determine sizing based on number of options
        let gridItemSizes;
        if (actions.length === 1) {
            gridItemSizes = { xs: 12 }
        }
        else if (actions.length === 2) {
            gridItemSizes = { xs: 6 }
        }
        else {
            gridItemSizes = { xs: 4 }
        }

        return actions.map(([label, Icon, disabled, isSubmit, onClick]: DialogActionItem, index) => (
            <Grid item key={label} {...gridItemSizes} p={1} sx={{ paddingTop: 0 }}>
                <Button
                    fullWidth
                    startIcon={<Icon />}
                    disabled={disabled}
                    type={isSubmit ? "submit" : "button"}
                    onClick={onClick}
                >{label}</Button>
            </Grid>
        ))
    }, [actions]);

    return (
        <Measure
            bounds
            onResize={handleResize}
        >
            {({ measureRef }) => (
                <Box sx={{
                    position: fixed ? 'fixed' : 'relative',
                    bottom: fixed ? '0' : 'auto',
                    width: fixed ? '-webkit-fill-available' : 'auto',
                    zIndex: 4,
                    background: palette.primary.main,
                    display: 'flex',
                    justifyContent: 'center',
                    alignItems: 'center',
                }}>
                    <Grid container ref={measureRef} spacing={2} sx={{
                        maxWidth: 'min(700px, 100%)',
                        margin: 0,
                    }}>
                        {gridItems}
                    </Grid>
                </Box>
            )}
        </Measure>
    )
}