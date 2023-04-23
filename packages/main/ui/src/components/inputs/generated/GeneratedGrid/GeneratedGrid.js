import { jsx as _jsx } from "react/jsx-runtime";
import { Grid, Stack } from "@mui/material";
import { useMemo } from "react";
import { ContentCollapse } from "../../../containers/ContentCollapse/ContentCollapse";
import { GeneratedInputComponent } from "../GeneratedInputComponent/GeneratedInputComponent";
export const generateGridItem = (component, index) => (_jsx(Grid, { item: true, children: component }, `grid-${index}`));
export const calculateGridItemSize = (numItems) => {
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
};
export const GeneratedGrid = ({ childContainers, fields, layout, onUpload, theme, zIndex, }) => {
    console.log("rendering grid", fields, childContainers);
    const { containers, splitFields } = useMemo(() => {
        console.log("rendering grid.containers, grid.splitfields");
        let splitFields = [];
        let containers;
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
        const grids = [];
        console.log("split fields", splitFields);
        for (let i = 0; i < splitFields.length; i++) {
            const currFields = splitFields[i];
            const currLayout = containers[i];
            const gridItems = currFields.map((fieldData, index) => {
                const inputComponent = _jsx(GeneratedInputComponent, { fieldData: fieldData, index: index, onUpload: onUpload, zIndex: zIndex });
                return inputComponent ? generateGridItem(inputComponent, index) : null;
            });
            const itemsContainer = _jsx(Grid, { container: true, direction: currLayout?.direction ?? "row", spacing: currLayout?.spacing ?? calculateGridItemSize(currFields.length), columnSpacing: currLayout?.columnSpacing, rowSpacing: currLayout?.rowSpacing, children: gridItems }, `form-container-${i}`);
            console.log("generatedGrid: item ifo:", currLayout);
            grids.push(_jsx("fieldset", { style: {
                    borderRadius: "8px",
                    border: currLayout?.showBorder ? `1px solid ${theme.palette.background.textPrimary}` : "none",
                }, children: currLayout?.title ? _jsx(ContentCollapse, { helpText: currLayout.description ?? undefined, title: currLayout.title, titleComponent: "legend", children: itemsContainer }) : itemsContainer }, `grid-container-${i}`));
        }
        return grids;
    }, [containers, onUpload, splitFields, theme.palette.background.textPrimary, zIndex]);
    return (_jsx(Stack, { direction: layout?.direction ?? "row", spacing: layout?.spacing, children: grids }, "form-container"));
};
//# sourceMappingURL=GeneratedGrid.js.map