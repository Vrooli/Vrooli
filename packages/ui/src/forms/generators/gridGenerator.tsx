import { Grid } from '@mui/material';
import { FieldData, GridContainer, GridContainerBase, GridSpacing, GridItemSpacing } from 'forms/types';
import { generateInputComponent } from '.';

/**
 * Wraps a component in a Grid item
 * @param component The component to wrap
 * @param sizes The sizes of the grid item
 * @param index The index of the grid
 * @returns Grid item component
 */
export const generateGridItem = (component: React.ReactElement, sizes: GridItemSpacing | undefined, index: number): React.ReactElement => (
    <Grid item key={`grid-${index}`} {...sizes}>
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
    console.log('generateGrid start', layout, childContainers, fields, formik);
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
    console.log('splitFields', splitFields);
    // Generate grid for each container
    let grids: React.ReactElement[] = [];
    for (let i = 0; i < splitFields.length; i++) {
        const currFields = splitFields[i];
        const currLayout = childContainers ? childContainers[i] : layout;
        console.log('in splitFields loop', currFields, currLayout);
        // Generate component for each field in the grid, and wrap it in a grid item
        const gridItems: Array<React.ReactElement | null> = currFields.map((field, index) => {
            const inputComponent = generateInputComponent({
                data: field,
                formik,
                index,
            });
            return inputComponent ? generateGridItem(inputComponent, currLayout?.itemSpacing, index) : null;
        });
        console.log('got gridItems', gridItems);
        grids.push(
            <Grid
                container
                direction={layout?.direction ?? 'row'}
                key={`grid-container-${i}`}
                spacing={layout?.spacing}
                columnSpacing={layout?.columnSpacing}
                rowSpacing={layout?.rowSpacing}
            >
                {gridItems}
            </Grid>
        )
    }
    console.log('finished generating grids', grids);
    return grids;
};