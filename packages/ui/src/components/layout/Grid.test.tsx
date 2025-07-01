import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Grid, GridFactory } from "./Grid.js";

describe("Grid Component", () => {
    describe("Basic rendering", () => {
        it("renders with default props", () => {
            render(<Grid data-testid="grid">Test content</Grid>);
            const grid = screen.getByTestId("grid");
            expect(grid).toBeDefined();
            expect(grid.tagName).toBe("DIV");
            expect(grid.textContent).toBe("Test content");
            expect(grid.classList.contains("tw-grid")).toBe(true);
        });

        it("renders with custom HTML element", () => {
            render(<Grid component="section" data-testid="grid">Section content</Grid>);
            const grid = screen.getByTestId("grid");
            expect(grid.tagName).toBe("SECTION");
            expect(grid.textContent).toBe("Section content");
        });

        it("applies custom className", () => {
            const customClass = "custom-grid-class";
            render(<Grid className={customClass} data-testid="grid">Content</Grid>);
            const grid = screen.getByTestId("grid");
            expect(grid.classList.contains(customClass)).toBe(true);
        });

        it("forwards ref correctly", () => {
            let refElement: HTMLElement | null = null;
            const TestComponent = () => (
                <Grid ref={(el) => { refElement = el; }} data-testid="grid">Content</Grid>
            );
            render(<TestComponent />);
            expect(refElement).toBeTruthy();
            expect(refElement?.tagName).toBe("DIV");
        });
    });

    describe("Grid columns", () => {
        it.each([
            [1, "tw-grid-cols-1"],
            [2, "tw-grid-cols-2"],
            [3, "tw-grid-cols-3"],
            [4, "tw-grid-cols-4"],
            [5, "tw-grid-cols-5"],
            [6, "tw-grid-cols-6"],
            [7, "tw-grid-cols-7"],
            [8, "tw-grid-cols-8"],
            [9, "tw-grid-cols-9"],
            [10, "tw-grid-cols-10"],
            [11, "tw-grid-cols-11"],
            [12, "tw-grid-cols-12"],
        ] as const)("applies %s columns", (columns, expectedClass) => {
            render(<Grid columns={columns} data-testid="grid">Content</Grid>);
            const grid = screen.getByTestId("grid");
            expect(grid.classList.contains(expectedClass)).toBe(true);
        });

        it("renders without column class when columns is 'none'", () => {
            render(<Grid columns="none" data-testid="grid">Content</Grid>);
            const grid = screen.getByTestId("grid");
            expect(Array.from(grid.classList).some(c => c.includes("tw-grid-cols"))).toBe(false);
        });
    });

    describe("Grid rows", () => {
        it.each([
            [1, "tw-grid-rows-1"],
            [2, "tw-grid-rows-2"],
            [3, "tw-grid-rows-3"],
            [4, "tw-grid-rows-4"],
            [5, "tw-grid-rows-5"],
            [6, "tw-grid-rows-6"],
        ] as const)("applies %s rows", (rows, expectedClass) => {
            render(<Grid rows={rows} data-testid="grid">Content</Grid>);
            const grid = screen.getByTestId("grid");
            expect(grid.classList.contains(expectedClass)).toBe(true);
        });

        it("renders without row class when rows is 'none'", () => {
            render(<Grid rows="none" data-testid="grid">Content</Grid>);
            const grid = screen.getByTestId("grid");
            expect(Array.from(grid.classList).some(c => c.includes("tw-grid-rows"))).toBe(false);
        });
    });

    describe("Grid gaps", () => {
        it.each([
            ["none", ""],
            ["xs", "tw-gap-1"],
            ["sm", "tw-gap-2"],
            ["md", "tw-gap-4"],
            ["lg", "tw-gap-6"],
            ["xl", "tw-gap-8"],
        ] as const)("applies %s gap", (gap, expectedClass) => {
            render(<Grid gap={gap} data-testid="grid">Content</Grid>);
            const grid = screen.getByTestId("grid");
            if (expectedClass) {
                expect(grid.classList.contains(expectedClass)).toBe(true);
            }
        });

        it("applies column gap separately", () => {
            render(<Grid columnGap="lg" data-testid="grid">Content</Grid>);
            const grid = screen.getByTestId("grid");
            expect(grid.classList.contains("tw-gap-x-6")).toBe(true);
        });

        it("applies row gap separately", () => {
            render(<Grid rowGap="sm" data-testid="grid">Content</Grid>);
            const grid = screen.getByTestId("grid");
            expect(grid.classList.contains("tw-gap-y-2")).toBe(true);
        });

        it("gap overrides column and row gaps", () => {
            render(<Grid gap="md" columnGap="lg" rowGap="sm" data-testid="grid">Content</Grid>);
            const grid = screen.getByTestId("grid");
            expect(grid.classList.contains("tw-gap-4")).toBe(true);
            expect(grid.classList.contains("tw-gap-x-6")).toBe(false);
            expect(grid.classList.contains("tw-gap-y-2")).toBe(false);
        });

        it("applies both column and row gaps when gap is not specified", () => {
            render(<Grid columnGap="lg" rowGap="sm" data-testid="grid">Content</Grid>);
            const grid = screen.getByTestId("grid");
            expect(grid.classList.contains("tw-gap-x-6")).toBe(true);
            expect(grid.classList.contains("tw-gap-y-2")).toBe(true);
        });
    });

    describe("Grid alignment", () => {
        it.each([
            ["start", "tw-items-start"],
            ["center", "tw-items-center"],
            ["end", "tw-items-end"],
            ["stretch", "tw-items-stretch"],
        ] as const)("applies %s align items", (alignItems, expectedClass) => {
            render(<Grid alignItems={alignItems} data-testid="grid">Content</Grid>);
            const grid = screen.getByTestId("grid");
            expect(grid.classList.contains(expectedClass)).toBe(true);
        });

        it.each([
            ["start", "tw-justify-start"],
            ["center", "tw-justify-center"],
            ["end", "tw-justify-end"],
            ["between", "tw-justify-between"],
            ["around", "tw-justify-around"],
            ["evenly", "tw-justify-evenly"],
        ] as const)("applies %s justify content", (justifyContent, expectedClass) => {
            render(<Grid justifyContent={justifyContent} data-testid="grid">Content</Grid>);
            const grid = screen.getByTestId("grid");
            expect(grid.classList.contains(expectedClass)).toBe(true);
        });
    });

    describe("Grid flow", () => {
        it.each([
            ["row", "tw-grid-flow-row"],
            ["col", "tw-grid-flow-col"],
            ["row-dense", "tw-grid-flow-row-dense"],
            ["col-dense", "tw-grid-flow-col-dense"],
        ] as const)("applies %s flow", (flow, expectedClass) => {
            render(<Grid flow={flow} data-testid="grid">Content</Grid>);
            const grid = screen.getByTestId("grid");
            expect(grid.classList.contains(expectedClass)).toBe(true);
        });
    });

    describe("Size options", () => {
        it("applies full width when specified", () => {
            render(<Grid fullWidth data-testid="grid">Content</Grid>);
            const grid = screen.getByTestId("grid");
            expect(grid.classList.contains("tw-w-full")).toBe(true);
        });

        it("applies full height when specified", () => {
            render(<Grid fullHeight data-testid="grid">Content</Grid>);
            const grid = screen.getByTestId("grid");
            expect(grid.classList.contains("tw-h-full")).toBe(true);
        });

        it("applies both full width and height", () => {
            render(<Grid fullWidth fullHeight data-testid="grid">Content</Grid>);
            const grid = screen.getByTestId("grid");
            expect(grid.classList.contains("tw-w-full")).toBe(true);
            expect(grid.classList.contains("tw-h-full")).toBe(true);
        });
    });

    describe("Content", () => {
        it("renders text content", () => {
            render(<Grid>Simple text content</Grid>);
            expect(screen.getByText("Simple text content")).toBeDefined();
        });

        it("renders multiple children", () => {
            render(
                <Grid>
                    <div data-testid="child-1">Child 1</div>
                    <div data-testid="child-2">Child 2</div>
                    <div data-testid="child-3">Child 3</div>
                </Grid>,
            );
            expect(screen.getByTestId("child-1")).toBeDefined();
            expect(screen.getByTestId("child-2")).toBeDefined();
            expect(screen.getByTestId("child-3")).toBeDefined();
        });

        it("renders empty content correctly", () => {
            render(<Grid data-testid="empty-grid"></Grid>);
            const grid = screen.getByTestId("empty-grid");
            expect(grid).toBeDefined();
            expect(grid.textContent).toBe("");
        });
    });

    describe("HTML attributes", () => {
        it("passes through HTML attributes", () => {
            render(
                <Grid
                    id="test-id"
                    role="region"
                    aria-label="Test grid"
                    data-testid="grid"
                >
                    Content
                </Grid>,
            );
            const grid = screen.getByTestId("grid");
            expect(grid.getAttribute("id")).toBe("test-id");
            expect(grid.getAttribute("role")).toBe("region");
            expect(grid.getAttribute("aria-label")).toBe("Test grid");
        });

        it("handles event handlers", () => {
            let clicked = false;
            const handleClick = () => { clicked = true; };
            
            render(<Grid onClick={handleClick} data-testid="clickable-grid">Click me</Grid>);
            const grid = screen.getByTestId("clickable-grid");
            grid.click();
            expect(clicked).toBe(true);
        });
    });

    describe("Factory components", () => {
        it("renders GridFactory.TwoColumn", () => {
            render(<GridFactory.TwoColumn data-testid="factory-two-col">Two column grid</GridFactory.TwoColumn>);
            const grid = screen.getByTestId("factory-two-col");
            expect(grid.classList.contains("tw-grid-cols-2")).toBe(true);
            expect(grid.classList.contains("tw-gap-4")).toBe(true);
        });

        it("renders GridFactory.ThreeColumn", () => {
            render(<GridFactory.ThreeColumn data-testid="factory-three-col">Three column grid</GridFactory.ThreeColumn>);
            const grid = screen.getByTestId("factory-three-col");
            expect(grid.classList.contains("tw-grid-cols-3")).toBe(true);
            expect(grid.classList.contains("tw-gap-4")).toBe(true);
        });

        it("renders GridFactory.FourColumn", () => {
            render(<GridFactory.FourColumn data-testid="factory-four-col">Four column grid</GridFactory.FourColumn>);
            const grid = screen.getByTestId("factory-four-col");
            expect(grid.classList.contains("tw-grid-cols-4")).toBe(true);
            expect(grid.classList.contains("tw-gap-4")).toBe(true);
        });

        it("renders GridFactory.TwelveColumn", () => {
            render(<GridFactory.TwelveColumn data-testid="factory-twelve-col">Twelve column grid</GridFactory.TwelveColumn>);
            const grid = screen.getByTestId("factory-twelve-col");
            expect(grid.classList.contains("tw-grid-cols-12")).toBe(true);
            expect(grid.classList.contains("tw-gap-4")).toBe(true);
        });

        it("renders GridFactory.Centered", () => {
            render(<GridFactory.Centered data-testid="factory-centered">Centered grid</GridFactory.Centered>);
            const grid = screen.getByTestId("factory-centered");
            expect(grid.classList.contains("tw-items-center")).toBe(true);
            expect(grid.classList.contains("tw-justify-center")).toBe(true);
        });

        it("renders GridFactory.SpacedEvenly", () => {
            render(<GridFactory.SpacedEvenly data-testid="factory-spaced">Spaced evenly grid</GridFactory.SpacedEvenly>);
            const grid = screen.getByTestId("factory-spaced");
            expect(grid.classList.contains("tw-justify-evenly")).toBe(true);
        });

        it("renders GridFactory.Responsive", () => {
            render(<GridFactory.Responsive data-testid="factory-responsive">Responsive grid</GridFactory.Responsive>);
            const grid = screen.getByTestId("factory-responsive");
            expect(grid.classList.contains("tw-w-full")).toBe(true);
            expect(grid.classList.contains("md:tw-grid-cols-2")).toBe(true);
            expect(grid.classList.contains("lg:tw-grid-cols-3")).toBe(true);
            expect(grid.classList.contains("xl:tw-grid-cols-4")).toBe(true);
        });
    });

    describe("Class name combinations", () => {
        it("combines multiple styling props correctly", () => {
            render(
                <Grid
                    columns={4}
                    rows={2}
                    gap="lg"
                    alignItems="center"
                    justifyContent="between"
                    flow="col"
                    fullWidth
                    className="custom-class"
                    data-testid="combined-grid"
                >
                    Combined styling
                </Grid>,
            );
            const grid = screen.getByTestId("combined-grid");
            
            // Grid template
            expect(grid.classList.contains("tw-grid-cols-4")).toBe(true);
            expect(grid.classList.contains("tw-grid-rows-2")).toBe(true);
            
            // Gap
            expect(grid.classList.contains("tw-gap-6")).toBe(true);
            
            // Alignment
            expect(grid.classList.contains("tw-items-center")).toBe(true);
            expect(grid.classList.contains("tw-justify-between")).toBe(true);
            
            // Flow
            expect(grid.classList.contains("tw-grid-flow-col")).toBe(true);
            
            // Full width
            expect(grid.classList.contains("tw-w-full")).toBe(true);
            
            // Custom class
            expect(grid.classList.contains("custom-class")).toBe(true);
        });
    });

    describe("Edge cases", () => {
        it("handles null children", () => {
            render(<Grid data-testid="null-children">{null}</Grid>);
            const grid = screen.getByTestId("null-children");
            expect(grid).toBeDefined();
        });

        it("handles undefined children", () => {
            render(<Grid data-testid="undefined-children">{undefined}</Grid>);
            const grid = screen.getByTestId("undefined-children");
            expect(grid).toBeDefined();
        });

        it("handles empty string children", () => {
            render(<Grid data-testid="empty-string">{""}</Grid>);
            const grid = screen.getByTestId("empty-string");
            expect(grid).toBeDefined();
        });

        it("handles zero as children", () => {
            render(<Grid data-testid="zero-children">{0}</Grid>);
            const grid = screen.getByTestId("zero-children");
            expect(grid).toBeDefined();
            expect(grid.textContent).toBe("0");
        });

        it("handles boolean false as children", () => {
            render(<Grid data-testid="false-children">{false}</Grid>);
            const grid = screen.getByTestId("false-children");
            expect(grid).toBeDefined();
            expect(grid.textContent).toBe("");
        });
    });

    describe("Integration scenarios", () => {
        it("works as a form grid", () => {
            render(
                <Grid component="form" columns={2} gap="md" data-testid="form-grid">
                    <input type="text" placeholder="First Name" />
                    <input type="text" placeholder="Last Name" />
                    <input type="email" placeholder="Email" />
                    <input type="phone" placeholder="Phone" />
                </Grid>,
            );
            const formGrid = screen.getByTestId("form-grid");
            expect(formGrid.tagName).toBe("FORM");
            expect(screen.getByPlaceholderText("First Name")).toBeDefined();
            expect(screen.getByPlaceholderText("Last Name")).toBeDefined();
            expect(screen.getByPlaceholderText("Email")).toBeDefined();
            expect(screen.getByPlaceholderText("Phone")).toBeDefined();
        });

        it("works with nested grids", () => {
            render(
                <Grid columns={2} gap="lg" data-testid="outer-grid">
                    <div>Left content</div>
                    <Grid columns={2} gap="sm" data-testid="inner-grid">
                        <div>Inner 1</div>
                        <div>Inner 2</div>
                    </Grid>
                </Grid>,
            );
            
            const outerGrid = screen.getByTestId("outer-grid");
            const innerGrid = screen.getByTestId("inner-grid");
            
            expect(outerGrid.contains(innerGrid)).toBe(true);
            expect(outerGrid.classList.contains("tw-grid-cols-2")).toBe(true);
            expect(innerGrid.classList.contains("tw-grid-cols-2")).toBe(true);
        });

        it("works with semantic elements", () => {
            render(
                <Grid component="main" columns={1} gap="lg" data-testid="main-grid">
                    <Grid component="header" columns={3} gap="md" data-testid="header-grid">
                        <div>Logo</div>
                        <div>Navigation</div>
                        <div>User Menu</div>
                    </Grid>
                    <Grid component="section" columns={2} gap="md" data-testid="content-grid">
                        <div>Main Content</div>
                        <div>Sidebar</div>
                    </Grid>
                    <Grid component="footer" columns={4} gap="sm" data-testid="footer-grid">
                        <div>Link 1</div>
                        <div>Link 2</div>
                        <div>Link 3</div>
                        <div>Link 4</div>
                    </Grid>
                </Grid>,
            );
            
            expect(screen.getByTestId("main-grid").tagName).toBe("MAIN");
            expect(screen.getByTestId("header-grid").tagName).toBe("HEADER");
            expect(screen.getByTestId("content-grid").tagName).toBe("SECTION");
            expect(screen.getByTestId("footer-grid").tagName).toBe("FOOTER");
        });

        it("works with responsive column counts", () => {
            render(
                <Grid
                    columns={1}
                    gap="md"
                    className="sm:tw-grid-cols-2 md:tw-grid-cols-3 lg:tw-grid-cols-4"
                    data-testid="responsive-grid"
                >
                    <div>Item 1</div>
                    <div>Item 2</div>
                    <div>Item 3</div>
                    <div>Item 4</div>
                </Grid>,
            );
            
            const grid = screen.getByTestId("responsive-grid");
            expect(grid.classList.contains("tw-grid-cols-1")).toBe(true);
            expect(grid.classList.contains("sm:tw-grid-cols-2")).toBe(true);
            expect(grid.classList.contains("md:tw-grid-cols-3")).toBe(true);
            expect(grid.classList.contains("lg:tw-grid-cols-4")).toBe(true);
        });
    });

    describe("Grid Items", () => {
        describe("Item mode detection", () => {
            it("renders as grid item when item prop is true", () => {
                render(<Grid item data-testid="grid-item">Grid item content</Grid>);
                const gridItem = screen.getByTestId("grid-item");
                expect(gridItem).toBeDefined();
                expect(gridItem.classList.contains("tw-grid")).toBe(false);
            });

            it("renders as grid container when item prop is false", () => {
                render(<Grid data-testid="grid-container">Grid container content</Grid>);
                const gridContainer = screen.getByTestId("grid-container");
                expect(gridContainer.classList.contains("tw-grid")).toBe(true);
            });
        });

        describe("Column spanning", () => {
            it.each([
                [1, "tw-col-span-1"],
                [2, "tw-col-span-2"],
                [3, "tw-col-span-3"],
                [4, "tw-col-span-4"],
                [5, "tw-col-span-5"],
                [6, "tw-col-span-6"],
                [7, "tw-col-span-7"],
                [8, "tw-col-span-8"],
                [9, "tw-col-span-9"],
                [10, "tw-col-span-10"],
                [11, "tw-col-span-11"],
                [12, "tw-col-span-12"],
            ] as const)("applies col-span-%s", (colSpan, expectedClass) => {
                render(<Grid item colSpan={colSpan} data-testid="grid-item">Content</Grid>);
                const gridItem = screen.getByTestId("grid-item");
                expect(gridItem.classList.contains(expectedClass)).toBe(true);
            });

            it("applies auto column span", () => {
                render(<Grid item colSpan="auto" data-testid="grid-item">Content</Grid>);
                const gridItem = screen.getByTestId("grid-item");
                expect(gridItem.classList.contains("tw-col-auto")).toBe(true);
            });

            it("applies full column span", () => {
                render(<Grid item colSpan="full" data-testid="grid-item">Content</Grid>);
                const gridItem = screen.getByTestId("grid-item");
                expect(gridItem.classList.contains("tw-col-span-full")).toBe(true);
            });
        });

        describe("Row spanning", () => {
            it.each([
                [1, "tw-row-span-1"],
                [2, "tw-row-span-2"],
                [3, "tw-row-span-3"],
                [4, "tw-row-span-4"],
                [5, "tw-row-span-5"],
                [6, "tw-row-span-6"],
            ] as const)("applies row-span-%s", (rowSpan, expectedClass) => {
                render(<Grid item rowSpan={rowSpan} data-testid="grid-item">Content</Grid>);
                const gridItem = screen.getByTestId("grid-item");
                expect(gridItem.classList.contains(expectedClass)).toBe(true);
            });
        });

        describe("Column positioning", () => {
            it.each([
                [1, "tw-col-start-1"],
                [2, "tw-col-start-2"],
                [3, "tw-col-start-3"],
                [4, "tw-col-start-4"],
                [5, "tw-col-start-5"],
                [6, "tw-col-start-6"],
            ] as const)("applies col-start-%s", (colStart, expectedClass) => {
                render(<Grid item colStart={colStart} data-testid="grid-item">Content</Grid>);
                const gridItem = screen.getByTestId("grid-item");
                expect(gridItem.classList.contains(expectedClass)).toBe(true);
            });

            it.each([
                [1, "tw-col-end-1"],
                [2, "tw-col-end-2"],
                [3, "tw-col-end-3"],
                [4, "tw-col-end-4"],
                [5, "tw-col-end-5"],
                [6, "tw-col-end-6"],
            ] as const)("applies col-end-%s", (colEnd, expectedClass) => {
                render(<Grid item colEnd={colEnd} data-testid="grid-item">Content</Grid>);
                const gridItem = screen.getByTestId("grid-item");
                expect(gridItem.classList.contains(expectedClass)).toBe(true);
            });
        });

        describe("Self alignment", () => {
            it.each([
                ["start", "tw-self-start"],
                ["center", "tw-self-center"],
                ["end", "tw-self-end"],
                ["stretch", "tw-self-stretch"],
            ] as const)("applies align-self %s", (alignSelf, expectedClass) => {
                render(<Grid item alignSelf={alignSelf} data-testid="grid-item">Content</Grid>);
                const gridItem = screen.getByTestId("grid-item");
                expect(gridItem.classList.contains(expectedClass)).toBe(true);
            });

            it.each([
                ["start", "tw-justify-self-start"],
                ["center", "tw-justify-self-center"],
                ["end", "tw-justify-self-end"],
                ["stretch", "tw-justify-self-stretch"],
            ] as const)("applies justify-self %s", (justifySelf, expectedClass) => {
                render(<Grid item justifySelf={justifySelf} data-testid="grid-item">Content</Grid>);
                const gridItem = screen.getByTestId("grid-item");
                expect(gridItem.classList.contains(expectedClass)).toBe(true);
            });
        });

        describe("Item factory components", () => {
            it("renders GridFactory.ItemHalf", () => {
                render(<GridFactory.ItemHalf data-testid="factory-half">Half width item</GridFactory.ItemHalf>);
                const gridItem = screen.getByTestId("factory-half");
                expect(gridItem.classList.contains("tw-col-span-6")).toBe(true);
            });

            it("renders GridFactory.ItemThird", () => {
                render(<GridFactory.ItemThird data-testid="factory-third">Third width item</GridFactory.ItemThird>);
                const gridItem = screen.getByTestId("factory-third");
                expect(gridItem.classList.contains("tw-col-span-4")).toBe(true);
            });

            it("renders GridFactory.ItemQuarter", () => {
                render(<GridFactory.ItemQuarter data-testid="factory-quarter">Quarter width item</GridFactory.ItemQuarter>);
                const gridItem = screen.getByTestId("factory-quarter");
                expect(gridItem.classList.contains("tw-col-span-3")).toBe(true);
            });

            it("renders GridFactory.ItemTwoThirds", () => {
                render(<GridFactory.ItemTwoThirds data-testid="factory-two-thirds">Two thirds width item</GridFactory.ItemTwoThirds>);
                const gridItem = screen.getByTestId("factory-two-thirds");
                expect(gridItem.classList.contains("tw-col-span-8")).toBe(true);
            });

            it("renders GridFactory.ItemFull", () => {
                render(<GridFactory.ItemFull data-testid="factory-full">Full width item</GridFactory.ItemFull>);
                const gridItem = screen.getByTestId("factory-full");
                expect(gridItem.classList.contains("tw-col-span-full")).toBe(true);
            });

            it("renders GridFactory.ItemCentered", () => {
                render(<GridFactory.ItemCentered data-testid="factory-centered">Centered item</GridFactory.ItemCentered>);
                const gridItem = screen.getByTestId("factory-centered");
                expect(gridItem.classList.contains("tw-self-center")).toBe(true);
                expect(gridItem.classList.contains("tw-justify-self-center")).toBe(true);
            });

            it("renders GridFactory.ItemSidebar", () => {
                render(<GridFactory.ItemSidebar data-testid="factory-sidebar">Sidebar content</GridFactory.ItemSidebar>);
                const gridItem = screen.getByTestId("factory-sidebar");
                expect(gridItem.classList.contains("tw-col-span-3")).toBe(true);
            });

            it("renders GridFactory.ItemMainContent", () => {
                render(<GridFactory.ItemMainContent data-testid="factory-main">Main content</GridFactory.ItemMainContent>);
                const gridItem = screen.getByTestId("factory-main");
                expect(gridItem.classList.contains("tw-col-span-9")).toBe(true);
            });
        });

        describe("Container and item integration", () => {
            it("works with grid items inside grid container", () => {
                render(
                    <Grid columns={4} gap="md" data-testid="parent-grid">
                        <Grid item colSpan={2} data-testid="item-1">Item 1</Grid>
                        <Grid item data-testid="item-2">Item 2</Grid>
                        <Grid item data-testid="item-3">Item 3</Grid>
                    </Grid>,
                );
                
                const parentGrid = screen.getByTestId("parent-grid");
                const item1 = screen.getByTestId("item-1");
                const item2 = screen.getByTestId("item-2");
                const item3 = screen.getByTestId("item-3");
                
                expect(parentGrid.classList.contains("tw-grid")).toBe(true);
                expect(parentGrid.contains(item1)).toBe(true);
                expect(parentGrid.contains(item2)).toBe(true);
                expect(parentGrid.contains(item3)).toBe(true);
                expect(item1.classList.contains("tw-col-span-2")).toBe(true);
            });

            it("prevents container classes on items", () => {
                render(
                    <Grid 
                        item 
                        columns={3} 
                        gap="md" 
                        colSpan={2}
                        data-testid="grid-item"
                    >
                        Item content
                    </Grid>,
                );
                
                const gridItem = screen.getByTestId("grid-item");
                expect(gridItem.classList.contains("tw-grid")).toBe(false);
                expect(gridItem.classList.contains("tw-grid-cols-3")).toBe(false);
                expect(gridItem.classList.contains("tw-gap-4")).toBe(false);
                expect(gridItem.classList.contains("tw-col-span-2")).toBe(true);
            });

            it("prevents item classes on containers", () => {
                render(
                    <Grid 
                        columns={3} 
                        gap="md" 
                        colSpan={2}
                        alignSelf="center"
                        data-testid="grid-container"
                    >
                        Container content
                    </Grid>,
                );
                
                const gridContainer = screen.getByTestId("grid-container");
                expect(gridContainer.classList.contains("tw-grid")).toBe(true);
                expect(gridContainer.classList.contains("tw-grid-cols-3")).toBe(true);
                expect(gridContainer.classList.contains("tw-gap-4")).toBe(true);
                expect(gridContainer.classList.contains("tw-col-span-2")).toBe(false);
                expect(gridContainer.classList.contains("tw-self-center")).toBe(false);
            });
        });
    });
});
