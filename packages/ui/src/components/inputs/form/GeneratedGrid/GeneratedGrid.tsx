import { Grid, GridSpacing, Stack } from "@mui/material";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse";
import { FormInputType, GridContainer } from "forms/types";
import { ReactNode, useMemo } from "react";
import { FormInput } from "../FormInput/FormInput";
import { GeneratedGridProps } from "../types";

/**
 * Wraps a component in a Grid item
 * @param component The component to wrap
 * @param sizes The sizes of the grid item
 * @param index The index of the grid
 * @returns Grid item component
 */
export function generateGridItem(component: ReactNode, index: number): ReactNode {
    return (
        <Grid item key={`grid-${index}`}>
            {component}
        </Grid>
    );
}

/**
 * Calculates size of grid item based on the number of items in the grid. 
 * 1 item is { xs: 12 }, 
 * 2 items is { xs: 12, sm: 6 },
 * 3 items is { xs: 12, sm: 6, md: 4 },
 * 4+ items is { xs: 12, sm: 6, md: 4, lg: 3 }
 * @returns Size of grid item
 */
export function calculateGridItemSize(numItems: number): { [key: string]: GridSpacing } {
    switch (numItems) {
        case 1:
            return { xs: 12 };
        // eslint-disable-next-line no-magic-numbers
        case 2:
            return { xs: 12, sm: 6 };
        // eslint-disable-next-line no-magic-numbers
        case 3:
            return { xs: 12, sm: 6, md: 4 };
        default:
            return { xs: 12, sm: 6, md: 4, lg: 3 };
    }
}

export function GeneratedGrid({
    childContainers,
    fields,
    layout,
    theme,
}: GeneratedGridProps) {
    console.log("rendering grid", fields, childContainers);
    const { containers, splitFields } = useMemo(() => {
        console.log("rendering grid.containers, grid.splitfields");
        // Split fields into which containers they belong to.
        // Represented by 2D array, where each sub-array represents a container.
        let splitFields: FormInputType[][] = [];
        let containers: GridContainer[];
        if (!childContainers) {
            splitFields = [fields];
            containers = [{
                ...layout,
                title: undefined,
                description: undefined,
                totalItems: fields.length,
                showBorder: false,
            }];
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
        return { containers, splitFields };
    }, [childContainers, fields, layout]);

    const grids = useMemo(() => {
        console.log("generating grid.grids");
        // Generate grid for each container
        const grids: ReactNode[] = [];
        console.log("split fields", splitFields);
        for (let i = 0; i < splitFields.length; i++) {
            const currFields: FormInputType[] = splitFields[i];
            const currLayout: GridContainer = containers[i];
            // Generate component for each field in the grid, and wrap it in a grid item
            const gridItems: ReactNode[] = currFields.map((fieldData, index) => {
                const inputComponent = <FormInput
                    fieldData={fieldData}
                    index={index}
                />;
                return inputComponent ? generateGridItem(inputComponent, index) : null;
            });
            const itemsContainer = <Grid
                container
                direction={currLayout?.direction ?? "row"}
                key={`form-container-${i}`}
                spacing={currLayout?.spacing ?? calculateGridItemSize(currFields.length)}
                columnSpacing={currLayout?.columnSpacing}
                rowSpacing={currLayout?.rowSpacing}
            >
                {gridItems}
            </Grid>;
            // Each container is represented as a fieldset
            console.log("generatedGrid: item ifo:", currLayout);
            grids.push(
                <fieldset
                    key={`grid-container-${i}`}
                    style={{
                        borderRadius: "8px",
                        border: currLayout?.showBorder ? `1px solid ${theme.palette.background.textPrimary}` : "none",
                    }}
                >
                    {/* If a title is provided, the items are wrapped in a collapsible container */}
                    {currLayout?.title ? <ContentCollapse
                        disableCollapse={currLayout.disableCollapse}
                        helpText={currLayout.description ?? undefined}
                        title={currLayout.title}
                        titleComponent="legend"
                    >
                        {itemsContainer}
                    </ContentCollapse> : itemsContainer}
                </fieldset>,
            );
        }
        return grids;
    }, [containers, splitFields, theme.palette.background.textPrimary]);

    return (
        <Stack
            direction={layout?.direction ?? "row"}
            key={"form-container"}
            spacing={layout?.spacing}
            sx={{ width: "100%" }}
        >
            {grids}
        </Stack>
    );
}
