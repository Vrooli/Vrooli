import { Grid, GridSpacing, Stack } from "@mui/material";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse";
import { FormInputType, GridContainer } from "forms/types";
import { ReactNode, useMemo } from "react";
import { FormInput } from "../FormInput/FormInput";
import { GeneratedGridProps } from "../types";

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

type GeneratedFormItemProps = {
    /** The child form input or form structure element */
    children: JSX.Element | null | undefined;
    /** The number of fields in the grid, for calculating grid item size */
    fieldsInGrid: number;
    /** Whether to wrap the child in a grid item */
    isInGrid: boolean;
}

/**
 * A wrapper for a form item that can be wrapped in a grid item
 */
export function GeneratedGridItem({
    children,
    fieldsInGrid,
    isInGrid,
}: GeneratedFormItemProps) {
    return isInGrid ? <Grid item {...calculateGridItemSize(fieldsInGrid)}>{children}</Grid> : children;
}

// TODO will eventually be replaced with FormRunView
export function GeneratedGrid({
    childContainers,
    fields,
    layout,
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
        for (let i = 0; i < splitFields.length; i++) {
            const currFields: FormInputType[] = splitFields[i];
            const currLayout: GridContainer = containers[i];
            // Use grid for horizontal layout, and stack for vertical layout
            const direction = currLayout?.direction ?? "column";
            const useGrid = direction === "row";
            // Generate component for each field in the grid, and wrap it in a grid item
            const gridItems: ReactNode[] = currFields.map((fieldData, index) => {
                return <GeneratedGridItem
                    key={`grid-item-${fieldData.id}`}
                    fieldsInGrid={currFields.length}
                    isInGrid={useGrid}
                >
                    <FormInput
                        fieldData={fieldData}
                        index={index}
                    />
                </GeneratedGridItem>;
            });
            const itemsContainer = direction === "row" ? <Grid
                container
                direction={currLayout?.direction ?? "column"}
                key={`form-container-${i}`}
                spacing={2}
            >
                {gridItems}
            </Grid> : <Stack
                direction={currLayout?.direction ?? "column"}
                key={`form-container-${i}`}
                spacing={2}
            >
                {gridItems}
            </Stack>;
            // Each container is represented as a fieldset
            console.log("generatedGrid: item ifo:", currLayout);
            grids.push(
                <fieldset key={`grid-container-${i}`} style={{ border: "none" }}>
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
    }, [containers, splitFields]);

    return (
        <Stack
            direction={layout?.direction ?? "row"}
            key={"form-container"}
            spacing={4}
            sx={{ width: "100%" }}
        >
            {grids}
        </Stack>
    );
}
