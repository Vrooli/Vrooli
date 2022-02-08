import { Button, Grid } from "@mui/material";
import { DialogActionsContainerProps, DialogActionItem } from "../types";
import { useCallback, useMemo } from "react";
import Measure from "react-measure";

export const DialogActionsContainer = ({
    actions,
    onResize,
}: DialogActionsContainerProps) => {

    const handleResize = useCallback(({ bounds }: any) => {
        onResize({ width: bounds.width, height: bounds.height });
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
            <Grid item key={label} {...gridItemSizes} sx={{ padding: 2 }}>
                <Button
                    fullWidth
                    startIcon={<Icon />}
                    disabled={disabled}
                    type={isSubmit ? "submit" : "button"}
                    onClick={() => onClick ? onClick() : undefined}
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
                    position: 'fixed', 
                    bottom: '0', 
                    background: (t) => t.palette.primary.main,
                    marginLeft: -1,
                }}>
                    {gridItems}
                </Grid>
            )}
        </Measure>
    )
}