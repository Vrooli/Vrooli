/* eslint-disable @typescript-eslint/ban-ts-comment */
import userEvent from "@testing-library/user-event";
import { render, screen, waitFor } from "../../__mocks__/testUtils";
import { NavigationArrows } from "./ChatBubble";

// describe("ChatBubbleStatus Component", () => {
//     // eslint-disable-next-line @typescript-eslint/no-unused-vars
//     let theme: Theme;

//     beforeAll(() => {
//         theme = themes[DEFAULT_THEME];
//         // eslint-disable-next-line @typescript-eslint/no-empty-function
//         jest.spyOn(console, "warn").mockImplementation(() => { });
//     });

//     afterAll(() => {
//         jest.restoreAllMocks();
//     });

//     beforeEach(() => {
//         jest.useFakeTimers();
//     });

//     afterEach(() => {
//         jest.runOnlyPendingTimers();
//         jest.useRealTimers();
//     });

//     it("should display CircularProgress with appropriate color when status is \"sending\"", () => {
//         // eslint-disable-next-line @typescript-eslint/no-empty-function
//         render(<ChatBubbleStatus status="sending" showButtons={false} isEditing={false} onRetry={() => { }} onEdit={() => { }} onDelete={() => { }} />);
//         const progress = screen.getByRole("progressbar");
//         expect(progress).toBeInTheDocument();
//         expect(progress).toHaveStyle(`color: ${theme.palette.secondary.main}`);
//     });

//     it("should increase progress over time when status is \"sending\"", () => {
//         // eslint-disable-next-line @typescript-eslint/no-empty-function
//         render(<ChatBubbleStatus status="sending" showButtons={false} isEditing={false} onRetry={() => { }} onEdit={() => { }} onDelete={() => { }} />);

//         act(() => {
//             jest.advanceTimersByTime(50); // Advance time by 50ms
//         });

//         const progress = screen.getByRole("progressbar");
//         expect(progress).toHaveAttribute("aria-valuenow", "3");
//     });

//     it("should reset progress after status changes from \"sending\" to \"sent\"", () => {
//         // eslint-disable-next-line @typescript-eslint/no-empty-function
//         const { rerender } = render(<ChatBubbleStatus status="sending" showButtons={false} isEditing={false} onRetry={() => { }} onEdit={() => { }} onDelete={() => { }} />);

//         act(() => {
//             jest.advanceTimersByTime(300); // Advance time by 300ms
//         });

//         // eslint-disable-next-line @typescript-eslint/no-empty-function
//         rerender(<ChatBubbleStatus status="sent" showButtons={false} isEditing={false} onRetry={() => { }} onEdit={() => { }} onDelete={() => { }} />);

//         act(() => {
//             jest.advanceTimersByTime(1000); // Wait for reset
//         });

//         expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();
//     });

//     it("should display ErrorIcon when status is \"failed\"", () => {
//         // eslint-disable-next-line @typescript-eslint/no-empty-function
//         render(<ChatBubbleStatus status="failed" showButtons={false} isEditing={false} onRetry={() => { }} onEdit={() => { }} onDelete={() => { }} />);
//         const errorButton = screen.getByLabelText("retry");
//         expect(errorButton).toBeInstanceOf(HTMLButtonElement);
//         // Allow any red shade
//         expect(errorButton).toHaveStyle(`color: ${red[500]}`);
//     });

//     it("should call onRetry when ErrorIcon is clicked", () => {
//         const onRetryMock = jest.fn();
//         // eslint-disable-next-line @typescript-eslint/no-empty-function
//         render(<ChatBubbleStatus status="failed" showButtons={false} isEditing={false} onRetry={onRetryMock} onEdit={() => { }} onDelete={() => { }} />);
//         const errorButton = screen.getByLabelText("retry");
//         fireEvent.click(errorButton);
//         expect(onRetryMock).toHaveBeenCalledTimes(1);
//     });

//     it("should call onEdit when EditIcon is clicked", () => {
//         const onEditMock = jest.fn();
//         // eslint-disable-next-line @typescript-eslint/no-empty-function
//         render(<ChatBubbleStatus status="sent" showButtons={true} isEditing={false} onRetry={() => { }} onEdit={onEditMock} onDelete={() => { }} />);
//         const editButton = screen.getByLabelText("edit");
//         fireEvent.click(editButton);
//         expect(onEditMock).toHaveBeenCalledTimes(1);
//     });

//     it("should call onDelete when DeleteIcon is clicked", () => {
//         const onDeleteMock = jest.fn();
//         // eslint-disable-next-line @typescript-eslint/no-empty-function
//         render(<ChatBubbleStatus status="sent" showButtons={true} isEditing={false} onRetry={() => { }} onEdit={() => { }} onDelete={onDeleteMock} />);
//         const deleteButton = screen.getByLabelText("delete");
//         fireEvent.click(deleteButton);
//         expect(onDeleteMock).toHaveBeenCalledTimes(1);
//     });

//     it("should display edit and delete buttons when showButtons is true and status is not \"sending\"", () => {
//         // eslint-disable-next-line @typescript-eslint/no-empty-function
//         render(<ChatBubbleStatus status="sent" showButtons={true} isEditing={false} onRetry={() => { }} onEdit={() => { }} onDelete={() => { }} />);
//         const editButton = screen.getByLabelText("edit");
//         const deleteButton = screen.getByLabelText("delete");
//         expect(editButton).toBeInTheDocument();
//         expect(deleteButton).toBeInTheDocument();
//     });

//     it("should not display any buttons when isEditing is true", () => {
//         // eslint-disable-next-line @typescript-eslint/no-empty-function
//         render(<ChatBubbleStatus status="sent" showButtons={true} isEditing={true} onRetry={() => { }} onEdit={() => { }} onDelete={() => { }} />);
//         expect(screen.queryByRole("button")).not.toBeInTheDocument();
//     });
// });

describe("NavigationArrows Component", () => {
    const handleIndexChangeMock = jest.fn();

    beforeAll(() => {
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        jest.spyOn(console, "warn").mockImplementation(() => { });
    });

    afterAll(() => {
        jest.restoreAllMocks();
    });

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
