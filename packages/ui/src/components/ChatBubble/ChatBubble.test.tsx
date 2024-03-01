import userEvent from "@testing-library/user-event";
import { render, screen, waitFor } from "../../__mocks__/testUtils";
import { NavigationArrows } from "./ChatBubble";

describe("NavigationArrows Component", () => {
    const handleIndexChangeMock = jest.fn();

    beforeEach(() => {
        jest.clearAllMocks();
    });

    it("should not render if there are less than 2 siblings", () => {
        render(<NavigationArrows activeIndex={0} numSiblings={1} onIndexChange={handleIndexChangeMock} />);
        expect(screen.queryByRole("button")).not.toBeInTheDocument();
    });

    it("should render both buttons disabled when there is no possibility to navigate", () => {
        render(<NavigationArrows activeIndex={0} numSiblings={1} onIndexChange={handleIndexChangeMock} />);
        expect(screen.queryAllByRole("button")).toHaveLength(0);
    });

    it("should enable the right button when there are more siblings ahead", async () => {
        render(<NavigationArrows activeIndex={0} numSiblings={2} onIndexChange={handleIndexChangeMock} />);
        const rightButton = screen.getByLabelText("right");
        expect(rightButton).toBeEnabled();
        userEvent.click(rightButton);
        await waitFor(() => {
            expect(handleIndexChangeMock).toHaveBeenCalledWith(1);
        });
    });

    it("should enable the left button when there are siblings behind", async () => {
        render(<NavigationArrows activeIndex={1} numSiblings={2} onIndexChange={handleIndexChangeMock} />);
        const leftButton = screen.getByLabelText("left");
        expect(leftButton).toBeEnabled();
        userEvent.click(leftButton);
        await waitFor(() => {
            expect(handleIndexChangeMock).toHaveBeenCalledWith(0);
        });
    });

    it("should display the current index and total siblings correctly", () => {
        const { rerender } = render(<NavigationArrows activeIndex={0} numSiblings={3} onIndexChange={handleIndexChangeMock} />);
        expect(screen.getByText("1/3")).toBeInTheDocument();

        rerender(<NavigationArrows activeIndex={1} numSiblings={3} onIndexChange={handleIndexChangeMock} />);
        expect(screen.getByText("2/3")).toBeInTheDocument();
    });

    it("should disable the left button when at the first sibling", () => {
        render(<NavigationArrows activeIndex={0} numSiblings={3} onIndexChange={handleIndexChangeMock} />);
        const leftButton = screen.getByLabelText("left");
        expect(leftButton).toBeDisabled();
    });

    it("should disable the right button when at the last sibling", () => {
        render(<NavigationArrows activeIndex={2} numSiblings={3} onIndexChange={handleIndexChangeMock} />);
        const rightButton = screen.getByLabelText("right");
        expect(rightButton).toBeDisabled();
    });
});
