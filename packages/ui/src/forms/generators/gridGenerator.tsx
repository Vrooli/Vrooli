import { Grid } from '@mui/material';
import { FieldData, GridContainer, GridContainerBase, GridSpacing, GridItemSpacing } from 'forms/types';
import { generateInputComponent } from '.';

/**
 * Wraps a component in a Grid item
 * @param component The component to wrap
 * @param sizes The sizes of the grid item
 * @returns Grid item component
 */
export const generateGridItem = (component: React.ReactElement, sizes: GridItemSpacing | undefined): React.ReactElement => (
    <Grid item {...sizes}>
        {component}
    </Grid>
);

/**
 * Wraps a list of Grid items in a Grid container
 * @returns Grid component
 * @param layout Layout data of the grid
 * @param fields Fields inside the grid
 * @param formik Formik object
 */
export const generateGrid = (
    layout: GridContainer | GridContainerBase | undefined,
    childContainers: GridContainer[] | undefined,
    fields: FieldData[],
    formik: any,
): React.ReactElement | React.ReactElement[] => {

    // Split fields into which containers they belong to.
    // Represented by 2D array, where each sub-array represents a container.
    let splitFields: FieldData[][] = [];
    if (!childContainers) splitFields = [fields];
    else {
        let lastField = 0;
        for (let i = 0; i < childContainers.length; i++) {
            const numInContainer = childContainers[i].totalItems;
            splitFields.push(fields.slice(lastField, lastField + numInContainer));
            lastField += numInContainer;
        }
    }

    // Generate grid for each container
    let grids: React.ReactElement[] = [];
    for (let i = 0; i < splitFields.length; i++) {
        const currFields = splitFields[i];
        const currLayout = childContainers ? childContainers[i] : layout;
        // Generate component for each field in the grid, and wrap it in a grid item
        const gridItems: Array<React.ReactElement | null> = currFields.map((field, index) => {
            const inputComponent = generateInputComponent(field, formik, index);
            return inputComponent ? generateGridItem(inputComponent, currLayout?.itemSpacing) : null;
        });
        grids.push(
            <Grid
                container
                spacing={layout?.spacing}
                columnSpacing={layout?.columnSpacing}
                rowSpacing={layout?.rowSpacing}
            >
                {gridItems}
            </Grid>
        )
    }
    return grids;
};