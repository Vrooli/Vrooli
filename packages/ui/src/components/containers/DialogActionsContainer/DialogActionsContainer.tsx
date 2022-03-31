import { Button, Grid } from "@mui/material";
import { DialogActionsContainerProps, DialogActionItem } from "../types";
import { useCallback, useMemo } from "react";
import Measure from "react-measure";

export const DialogActionsContainer = ({
    fixed = true,
    actions,
    onResize,
}: DialogActionsContainerProps) => {

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
            gridItemSizes = { xs: 12, sm: 6 }
        }
        else {
            gridItemSizes = { xs: 12, sm: 6, md: 4 }
        }

        return actions.map(([label, Icon, disabled, isSubmit, onClick]: DialogActionItem, index) => (
            <Grid item key={label} {...gridItemSizes} p={1} sx={{paddingTop: 0}}>
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
                <Grid container ref={measureRef} spacing={2} sx={{
                    position: fixed ? 'fixed' : 'relative',
                    bottom: fixed ? '0' : 'auto',
                    background: (t) => t.palette.primary.main,
                    margin: 0,
                    zIndex: 4,
                    width: fixed ? '-webkit-fill-available' : 'auto',
                }}>
                    {gridItems}
                </Grid>
            )}
        </Measure>
    )
}