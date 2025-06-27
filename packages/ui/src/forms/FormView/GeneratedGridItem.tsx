import { Grid } from "../../components/layout/Grid.js";
import type { GeneratedFormItemProps } from "./FormView.types.js";
import { calculateGridItemSize } from "./FormView.utils.js";

/**
 * A wrapper for a form item that can be wrapped in a grid item
 */
export function GeneratedGridItem({
    children,
    fieldsInGrid,
    isInGrid,
}: GeneratedFormItemProps) {
    return isInGrid ? (
        <Grid
            item
            {...calculateGridItemSize(fieldsInGrid)}
            data-testid="generated-grid-item"
            data-in-grid="true"
            data-fields-count={fieldsInGrid}
        >
            {children}
        </Grid>
    ) : (
        <div data-testid="generated-grid-item" data-in-grid="false">
            {children}
        </div>
    );
}
