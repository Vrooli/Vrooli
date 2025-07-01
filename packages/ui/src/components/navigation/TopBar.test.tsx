import { render, screen } from "@testing-library/react";
import React from "react";
import { describe, expect, it, vi } from "vitest";
import { ViewDisplayType } from "../../types.js";
import { TopBar } from "./TopBar.js";

// Mock the dependencies
vi.mock("../dialogs/DialogTitle/DialogTitle.js", () => ({
    DialogTitle: React.forwardRef(function MockDialogTitle(props: any, ref) {
        return (
            <div 
                ref={ref}
                data-testid={props["data-testid"]}
                data-display-type={props["data-display-type"]}
                data-title={props.title}
                data-help={props.help}
                data-title-id={props.id}
            >
                Dialog Title: {props.title}
            </div>
        );
    }),
}));

vi.mock("./Navbar.js", () => ({
    Navbar: function MockNavbar(props: any) {
        return (
            <div 
                data-testid={props["data-testid"]}
                data-display-type={props["data-display-type"]}
                data-title={props.title}
                data-help={props.help}
                data-title-behavior={props.titleBehavior}
            >
                Navbar: {props.title}
            </div>
        );
    },
}));

vi.mock("./PartialNavbar.js", () => ({
    PartialNavbar: function MockPartialNavbar(props: any) {
        return (
            <div 
                data-testid={props["data-testid"]}
                data-display-type={props["data-display-type"]}
                data-title={props.title}
                data-help={props.help}
            >
                Partial Navbar: {props.title}
            </div>
        );
    },
}));

vi.mock("../../utils/codes.js", () => ({
    randomString: () => "mock-random-id",
}));

describe("TopBar", () => {
    describe("Display type routing", () => {
        it("renders DialogTitle when display is Dialog", () => {
            render(
                <TopBar 
                    display={ViewDisplayType.Dialog}
                    title="Test Dialog Title"
                    help="Test help text"
                />,
            );

            const dialogTitle = screen.getByTestId("topbar-dialog");
            expect(dialogTitle).toBeDefined();
            expect(dialogTitle.getAttribute("data-display-type")).toBe(ViewDisplayType.Dialog);
            expect(dialogTitle.getAttribute("data-title")).toBe("Test Dialog Title");
            expect(dialogTitle.getAttribute("data-help")).toBe("Test help text");
            expect(dialogTitle.getAttribute("data-title-id")).toBe("mock-random-id");
            
            // Should not render other components
            expect(screen.queryByTestId("topbar-page")).toBeNull();
            expect(screen.queryByTestId("topbar-partial")).toBeNull();
        });

        it("renders PartialNavbar when display is Partial", () => {
            render(
                <TopBar 
                    display={ViewDisplayType.Partial}
                    title="Test Partial Title"
                    help="Test help text"
                />,
            );

            const partialNavbar = screen.getByTestId("topbar-partial");
            expect(partialNavbar).toBeDefined();
            expect(partialNavbar.getAttribute("data-display-type")).toBe(ViewDisplayType.Partial);
            expect(partialNavbar.getAttribute("data-title")).toBe("Test Partial Title");
            expect(partialNavbar.getAttribute("data-help")).toBe("Test help text");
            
            // Should not render other components
            expect(screen.queryByTestId("topbar-dialog")).toBeNull();
            expect(screen.queryByTestId("topbar-page")).toBeNull();
        });

        it("renders Navbar when display is Page", () => {
            render(
                <TopBar 
                    display={ViewDisplayType.Page}
                    title="Test Page Title"
                    help="Test help text"
                    titleBehavior="Show"
                />,
            );

            const navbar = screen.getByTestId("topbar-page");
            expect(navbar).toBeDefined();
            expect(navbar.getAttribute("data-display-type")).toBe(ViewDisplayType.Page);
            expect(navbar.getAttribute("data-title")).toBe("Test Page Title");
            expect(navbar.getAttribute("data-help")).toBe("Test help text");
            expect(navbar.getAttribute("data-title-behavior")).toBe("Show");
            
            // Should not render other components
            expect(screen.queryByTestId("topbar-dialog")).toBeNull();
            expect(screen.queryByTestId("topbar-partial")).toBeNull();
        });

        it("defaults to Navbar when display type is not Dialog or Partial", () => {
            render(
                <TopBar 
                    display="UnknownType" as any
                    title="Default Title"
                />,
            );

            const navbar = screen.getByTestId("topbar-page");
            expect(navbar).toBeDefined();
            expect(navbar.getAttribute("data-display-type")).toBe("UnknownType");
            expect(navbar.getAttribute("data-title")).toBe("Default Title");
            
            // Should not render other components
            expect(screen.queryByTestId("topbar-dialog")).toBeNull();
            expect(screen.queryByTestId("topbar-partial")).toBeNull();
        });
    });

    describe("Props passing for Dialog display", () => {
        it("passes title props to DialogTitle", () => {
            render(
                <TopBar 
                    display={ViewDisplayType.Dialog}
                    title="Dialog Test"
                    help="Dialog help"
                    titleId="custom-title-id"
                />,
            );

            const dialogTitle = screen.getByTestId("topbar-dialog");
            expect(dialogTitle.getAttribute("data-title")).toBe("Dialog Test");
            expect(dialogTitle.getAttribute("data-help")).toBe("Dialog help");
            expect(dialogTitle.getAttribute("data-title-id")).toBe("custom-title-id");
        });

        it("generates random titleId when not provided for Dialog", () => {
            render(
                <TopBar 
                    display={ViewDisplayType.Dialog}
                    title="Dialog Without ID"
                />,
            );

            const dialogTitle = screen.getByTestId("topbar-dialog");
            expect(dialogTitle.getAttribute("data-title-id")).toBe("mock-random-id");
        });

        it("passes all titleData props to DialogTitle", () => {
            const iconInfo = { name: "TestIcon" };
            const options = [
                { iconInfo: { name: "Option1" }, label: "Option 1", onClick: vi.fn() },
            ];

            render(
                <TopBar 
                    display={ViewDisplayType.Dialog}
                    title="Full Props Test"
                    help="Full help"
                    iconInfo={iconInfo}
                    options={options}
                    titleId="full-test-id"
                />,
            );

            const dialogTitle = screen.getByTestId("topbar-dialog");
            expect(dialogTitle).toBeDefined();
            expect(dialogTitle.getAttribute("data-title")).toBe("Full Props Test");
            expect(dialogTitle.getAttribute("data-help")).toBe("Full help");
            expect(dialogTitle.getAttribute("data-title-id")).toBe("full-test-id");
        });
    });

    describe("Props passing for Partial display", () => {
        it("passes all titleData props to PartialNavbar except titleBehavior", () => {
            const iconInfo = { name: "TestIcon" };
            const options = [
                { iconInfo: { name: "Option1" }, label: "Option 1", onClick: vi.fn() },
            ];

            render(
                <TopBar 
                    display={ViewDisplayType.Partial}
                    title="Partial Test"
                    help="Partial help"
                    iconInfo={iconInfo}
                    options={options}
                    titleBehavior="Hide"
                />,
            );

            const partialNavbar = screen.getByTestId("topbar-partial");
            expect(partialNavbar).toBeDefined();
            expect(partialNavbar.getAttribute("data-title")).toBe("Partial Test");
            expect(partialNavbar.getAttribute("data-help")).toBe("Partial help");
            // titleBehavior should not be passed to PartialNavbar
            expect(partialNavbar.getAttribute("data-title-behavior")).toBeNull();
        });
    });

    describe("Props passing for Page display", () => {
        it("passes titleBehavior and all titleData props to Navbar", () => {
            const iconInfo = { name: "TestIcon" };
            const options = [
                { iconInfo: { name: "Option1" }, label: "Option 1", onClick: vi.fn() },
            ];

            render(
                <TopBar 
                    display={ViewDisplayType.Page}
                    title="Page Test"
                    help="Page help"
                    titleBehavior="Hide"
                    iconInfo={iconInfo}
                    options={options}
                />,
            );

            const navbar = screen.getByTestId("topbar-page");
            expect(navbar).toBeDefined();
            expect(navbar.getAttribute("data-title")).toBe("Page Test");
            expect(navbar.getAttribute("data-help")).toBe("Page help");
            expect(navbar.getAttribute("data-title-behavior")).toBe("Hide");
        });

        it("works without titleBehavior prop", () => {
            render(
                <TopBar 
                    display={ViewDisplayType.Page}
                    title="Page Without Behavior"
                />,
            );

            const navbar = screen.getByTestId("topbar-page");
            expect(navbar).toBeDefined();
            expect(navbar.getAttribute("data-title")).toBe("Page Without Behavior");
            expect(navbar.getAttribute("data-title-behavior")).toBeNull();
        });
    });

    describe("Ref forwarding", () => {
        it("forwards ref to DialogTitle when display is Dialog", () => {
            const ref = React.createRef<HTMLElement>();
            
            render(
                <TopBar 
                    ref={ref}
                    display={ViewDisplayType.Dialog}
                    title="Ref Test"
                />,
            );

            expect(ref.current).toBeDefined();
            expect(ref.current?.getAttribute("data-testid")).toBe("topbar-dialog");
        });

        it("does not forward ref when display is not Dialog", () => {
            const ref = React.createRef<HTMLElement>();
            
            render(
                <TopBar 
                    ref={ref}
                    display={ViewDisplayType.Page}
                    title="No Ref Test"
                />,
            );

            // For Page and Partial displays, ref is not forwarded since they don't support it
            // This is expected behavior based on the component implementation
            expect(ref.current).toBeNull();
        });
    });

    describe("Display name", () => {
        it("has correct display name", () => {
            expect(TopBar.displayName).toBe("TopBar");
        });
    });

    describe("Edge cases", () => {
        it("handles minimal props for Dialog", () => {
            render(<TopBar display={ViewDisplayType.Dialog} />);
            
            const dialogTitle = screen.getByTestId("topbar-dialog");
            expect(dialogTitle).toBeDefined();
            expect(dialogTitle.getAttribute("data-title")).toBeNull();
            expect(dialogTitle.getAttribute("data-title-id")).toBe("mock-random-id");
        });

        it("handles minimal props for Partial", () => {
            render(<TopBar display={ViewDisplayType.Partial} />);
            
            const partialNavbar = screen.getByTestId("topbar-partial");
            expect(partialNavbar).toBeDefined();
            expect(partialNavbar.getAttribute("data-title")).toBeNull();
        });

        it("handles minimal props for Page", () => {
            render(<TopBar display={ViewDisplayType.Page} />);
            
            const navbar = screen.getByTestId("topbar-page");
            expect(navbar).toBeDefined();
            expect(navbar.getAttribute("data-title")).toBeNull();
            expect(navbar.getAttribute("data-title-behavior")).toBeNull();
        });

        it("handles empty string display type", () => {
            render(<TopBar display="" as any />);
            
            // Should default to Navbar
            const navbar = screen.getByTestId("topbar-page");
            expect(navbar).toBeDefined();
        });
    });

    describe("State transitions", () => {
        it("switches between different display types", () => {
            const { rerender } = render(
                <TopBar 
                    display={ViewDisplayType.Dialog}
                    title="Initial Dialog"
                />,
            );

            // Start with Dialog
            expect(screen.getByTestId("topbar-dialog")).toBeDefined();
            expect(screen.queryByTestId("topbar-page")).toBeNull();
            expect(screen.queryByTestId("topbar-partial")).toBeNull();

            // Switch to Page
            rerender(
                <TopBar 
                    display={ViewDisplayType.Page}
                    title="Now Page"
                />,
            );

            expect(screen.queryByTestId("topbar-dialog")).toBeNull();
            expect(screen.getByTestId("topbar-page")).toBeDefined();
            expect(screen.queryByTestId("topbar-partial")).toBeNull();

            // Switch to Partial
            rerender(
                <TopBar 
                    display={ViewDisplayType.Partial}
                    title="Now Partial"
                />,
            );

            expect(screen.queryByTestId("topbar-dialog")).toBeNull();
            expect(screen.queryByTestId("topbar-page")).toBeNull();
            expect(screen.getByTestId("topbar-partial")).toBeDefined();
        });

        it("maintains props consistency across state changes", () => {
            const { rerender } = render(
                <TopBar 
                    display={ViewDisplayType.Dialog}
                    title="Consistent Title"
                    help="Consistent Help"
                />,
            );

            let element = screen.getByTestId("topbar-dialog");
            expect(element.getAttribute("data-title")).toBe("Consistent Title");
            expect(element.getAttribute("data-help")).toBe("Consistent Help");

            // Switch to Page with same props
            rerender(
                <TopBar 
                    display={ViewDisplayType.Page}
                    title="Consistent Title"
                    help="Consistent Help"
                />,
            );

            element = screen.getByTestId("topbar-page");
            expect(element.getAttribute("data-title")).toBe("Consistent Title");
            expect(element.getAttribute("data-help")).toBe("Consistent Help");
        });
    });
});
