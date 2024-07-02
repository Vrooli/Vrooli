import { noop } from "@local/shared";
import { Grid, GridSpacing, Stack } from "@mui/material";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse";
import { FormInputType, FormSchema, GridContainer } from "forms/types";
import { ReactNode, useMemo } from "react";
import { FormInput } from "../FormInput/FormInput";

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

export type NormalizedGridContainer = Omit<GridContainer, "totalItems"> & {
    startIndex: number;
    endIndex: number;
}

/**
 * Function to convert a FormSchema into an array of containers with start and end indices for rendering.
 * @param formSchema The schema defining the form layout, containers, and elements.
 * @returns An array of containers with start and end indices for the elements in each container. 
 * If no containers are provided, returns one container covering all elements. 
 * Ensures valid ranges for each container.
 */
export function normalizeFormContainers(
    formSchema: Pick<FormSchema, "containers" | "elements">,
): NormalizedGridContainer[] {
    const { elements, containers } = formSchema;

    if (!Array.isArray(elements) || elements.length === 0) {
        // If there are no elements, return an empty array
        return [];
    }

    if (!Array.isArray(containers) || containers.length === 0) {
        // If no containers are provided, return one container for all elements
        return [{
            startIndex: 0,
            endIndex: elements.length - 1,
            disableCollapse: false,
        }];
    }

    const containerRanges: NormalizedGridContainer[] = [];
    let currentIndex = 0;

    containers.forEach(container => {
        if (currentIndex >= elements.length) {
            // If the current index is beyond the elements length, break the loop
            return;
        }

        const endIndex = Math.min(currentIndex + container.totalItems - 1, elements.length - 1);

        containerRanges.push({
            ...container,
            startIndex: currentIndex,
            endIndex,
        });

        currentIndex = endIndex + 1;
    });

    // Ensure all elements are covered in case totalItems were more than elements length
    if (currentIndex < elements.length) {
        containerRanges.push({
            startIndex: currentIndex,
            endIndex: elements.length - 1,
            disableCollapse: false,
        });
    }

    return containerRanges;
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
    containers,
    elements,
    layout,
}: FormSchema) {
    const sections = useMemo(() => {
        // Normalize/heal containers to ensure they cover all elements
        const normalizedContainers = normalizeFormContainers({ containers, elements });
        // Render each container as a stack or grid, depending on configuration
        const sections: ReactNode[] = [];
        for (let i = 0; i < normalizedContainers.length; i++) {
            const currContainer: NormalizedGridContainer = normalizedContainers[i];
            // Use grid for horizontal layout, and stack for vertical layout
            const direction = currContainer?.direction ?? "column";
            const useGrid = direction === "row";
            // Generate component for each field in the grid
            const gridItems: ReactNode[] = [];
            for (let j = currContainer.startIndex; j <= currContainer.endIndex; j++) {
                const fieldData = elements[j] as FormInputType;
                gridItems.push(<GeneratedGridItem
                    key={`grid-item-${fieldData.id}`}
                    fieldsInGrid={currContainer.endIndex - currContainer.startIndex + 1}
                    isInGrid={useGrid}
                >
                    <FormInput
                        disabled={false}
                        fieldData={fieldData}
                        index={j}
                        isEditing={false}
                        onConfigUpdate={noop}
                        onDelete={noop}
                    />
                </GeneratedGridItem>);
            }
            const itemsContainer = direction === "row" ? <Grid
                container
                direction="row"
                key={`form-container-${i}`}
                spacing={2}
            >
                {gridItems}
            </Grid> : <Stack
                direction="column"
                key={`form-container-${i}`}
                spacing={2}
            >
                {gridItems}
            </Stack>;
            // Each section is represented using a fieldset
            sections.push(
                <fieldset key={`grid-container-${i}`} style={{ border: "none" }}>
                    {/* If a title is provided, the items are wrapped in a collapsible container */}
                    {currContainer?.title ? <ContentCollapse
                        disableCollapse={currContainer.disableCollapse}
                        helpText={currContainer.description ?? undefined}
                        title={currContainer.title}
                        titleComponent="legend"
                    >
                        {itemsContainer}
                    </ContentCollapse> : itemsContainer}
                </fieldset>,
            );
        }
        return sections;
    }, [containers, elements]);

    return (
        <Stack
            direction={"column"}
            key={"form-container"}
            spacing={4}
            sx={{ width: "100%" }}
        >
            {sections}
        </Stack>
    );
}
