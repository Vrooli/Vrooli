import { Grid, GridSpacing, Stack, Theme } from '@mui/material';
import { Session } from '@shared/consts';
import { ContentCollapse } from 'components';
import { FieldData, GridContainer, GridContainerBase } from 'forms/types';
import { generateInputComponent } from '.';

/**
 * Wraps a component in a Grid item
 * @param component The component to wrap
 * @param sizes The sizes of the grid item
 * @param index The index of the grid
 * @returns Grid item component
 */
export const generateGridItem = (component: React.ReactElement, index: number): React.ReactElement => (
    <Grid item key={`grid-${index}`}>
        {component}
    </Grid>
);

/**
 * Calculates size of grid item based on the number of items in the grid. 
 * 1 item is { xs: 12 }, 
 * 2 items is { xs: 12, sm: 6 },
 * 3 items is { xs: 12, sm: 6, md: 4 },
 * 4+ items is { xs: 12, sm: 6, md: 4, lg: 3 }
 * @returns Size of grid item
 */
export const calculateGridItemSize = (numItems: number): { [key: string]: GridSpacing } => {
    switch (numItems) {
        case 1:
            return { xs: 12 };
        case 2:
            return { xs: 12, sm: 6 };
        case 3:
            return { xs: 12, sm: 6, md: 4 };
        default:
            return { xs: 12, sm: 6, md: 4, lg: 3 };
    }
}

interface GenerateGridProps {
    childContainers?: GridContainer[];
    fields: FieldData[];
    formik: any;
    layout?: GridContainer | GridContainerBase;
    onUpload: (fieldName: string, files: string[]) => void;
    session: Session;
    theme: Theme;
    zIndex: number;
}
/**
 * Wraps a list of Grid items in a Grid container
 * @returns Grid component
 */
export const generateGrid = ({
    childContainers,
    fields,
    formik,
    layout,
    onUpload,
    session,
    theme,
    zIndex,
}: GenerateGridProps): JSX.Element => {
    // Split fields into which containers they belong to.
    // Represented by 2D array, where each sub-array represents a container.
    let splitFields: FieldData[][] = [];
    let containers: GridContainer[];
    if (!childContainers) {
        splitFields = [fields];
        containers = [{
            ...layout,
            title: undefined,
            description: undefined,
            totalItems: fields.length,
            showBorder: false,
        }]
    }
    else {
        let lastField = 0;
        for (let i = 0; i < childContainers.length; i++) {
            const numInContainer = childContainers[i].totalItems;
            splitFields.push(fields.slice(lastField, lastField + numInContainer));
            lastField += numInContainer;
        }
        containers = childContainers;
    }
    // Generate grid for each container
    let grids: React.ReactElement[] = [];
    console.log('split fields', splitFields);
    for (let i = 0; i < splitFields.length; i++) {
        const currFields: FieldData[] = splitFields[i];
        const currLayout: GridContainer = containers[i];
        // Generate component for each field in the grid, and wrap it in a grid item
        const gridItems: Array<React.ReactElement | null> = currFields.map((fieldData, index) => {
            const inputComponent = generateInputComponent({
                formik,
                fieldData,
                index,
                onUpload,
                session,
                zIndex,
            });
            return inputComponent ? generateGridItem(inputComponent, index) : null;
        });
        const itemsContainer = <Grid
            container
            direction={currLayout?.direction ?? 'row'}
            key={`form-container-${i}`}
            spacing={currLayout?.spacing ?? calculateGridItemSize(currFields.length)}
            columnSpacing={currLayout?.columnSpacing}
            rowSpacing={currLayout?.rowSpacing}
        >
            {gridItems}
        </Grid>
        // Each container is represented as a fieldset
        grids.push(
            <fieldset
                key={`grid-container-${i}`}
                style={{
                    borderRadius: '8px',
                    border: currLayout?.showBorder ? `1px solid ${theme.palette.background.textPrimary}` : 'none',
                }}
            >
                {/* If a title is provided, the items are wrapped in a collapsible container */}
                {currLayout?.title ? <ContentCollapse
                    helpText={currLayout.description ?? undefined}
                    title={currLayout.title}
                    titleComponent="legend"
                >
                    {itemsContainer}
                </ContentCollapse> : itemsContainer}
            </fieldset>
        )
    }
    return (
        <Stack
            direction={layout?.direction ?? 'row'}
            key={`form-container`}
            spacing={layout?.spacing}
        >
            {grids}
        </Stack>
    )
};