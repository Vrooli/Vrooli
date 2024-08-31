import { act, renderHook } from "@testing-library/react";
import { useInfiniteScroll } from "./useInfiniteScroll";

describe("useInfiniteScroll", () => {
    let mockLoadMore: jest.Mock;
    let mockAddEventListener: jest.Mock;
    let mockRemoveEventListener: jest.Mock;
    let mockScrollContainer: {
        addEventListener: jest.Mock;
        removeEventListener: jest.Mock;
        scrollTop: number;
        scrollHeight: number;
        clientHeight: number;
    };

    beforeEach(() => {
        mockLoadMore = jest.fn();
        mockAddEventListener = jest.fn();
        mockRemoveEventListener = jest.fn();
        mockScrollContainer = {
            addEventListener: mockAddEventListener,
            removeEventListener: mockRemoveEventListener,
            scrollTop: 0,
            // Scrollable distance is calculated as scrollHeight - clientHeight, so the 
            // effective scrollable distance is 1000px
            scrollHeight: 1500,
            clientHeight: 500,
        };

        // Mock getElementById
        document.getElementById = jest.fn().mockImplementation(() => mockScrollContainer);
    });

    afterEach(() => {
        jest.resetAllMocks();
    });

    it("should add scroll event listener on mount", () => {
        renderHook(() => useInfiniteScroll({
            loading: false,
            loadMore: mockLoadMore,
            scrollContainerId: "test-container",
        }));

        expect(document.getElementById).toHaveBeenCalledWith("test-container");
        expect(mockAddEventListener).toHaveBeenCalledWith("scroll", expect.any(Function));
    });

    it("should remove scroll event listener on unmount", () => {
        const { unmount } = renderHook(() => useInfiniteScroll({
            loading: false,
            loadMore: mockLoadMore,
            scrollContainerId: "test-container",
        }));

        unmount();

        expect(mockRemoveEventListener).toHaveBeenCalledWith("scroll", expect.any(Function));
    });

    it("should not call loadMore when loading is true", () => {
        renderHook(() => useInfiniteScroll({
            loading: true,
            loadMore: mockLoadMore,
            scrollContainerId: "test-container",
        }));

        const scrollHandler = mockAddEventListener.mock.calls[0][1];
        act(() => {
            scrollHandler();
        });

        expect(mockLoadMore).not.toHaveBeenCalled();
    });

    it("should call loadMore when scrolled past threshold", () => {
        renderHook(() => useInfiniteScroll({
            loading: false,
            loadMore: mockLoadMore,
            scrollContainerId: "test-container",
        }));

        const scrollHandler = mockAddEventListener.mock.calls[0][1];

        // Simulate scrolling past threshold
        mockScrollContainer.scrollTop = 600;

        act(() => {
            scrollHandler();
        });

        expect(mockLoadMore).toHaveBeenCalled();
    });

    it("should not call loadMore when not scrolled past threshold", () => {
        renderHook(() => useInfiniteScroll({
            loading: false,
            loadMore: mockLoadMore,
            scrollContainerId: "test-container",
        }));

        const scrollHandler = mockAddEventListener.mock.calls[0][1];

        // Simulate scrolling, but not past threshold
        mockScrollContainer.scrollTop = 400;

        act(() => {
            scrollHandler();
        });

        expect(mockLoadMore).not.toHaveBeenCalled();
    });

    it("should use custom threshold when provided", () => {
        renderHook(() => useInfiniteScroll({
            loading: false,
            loadMore: mockLoadMore,
            scrollContainerId: "test-container",
            threshold: 900, // Since the effective scrollable distance is 1000px, this should trigger loadMore when scrolled past 100px
        }));

        const scrollHandler = mockAddEventListener.mock.calls[0][1];

        // Simulate scrolling past custom threshold
        mockScrollContainer.scrollTop = 101;

        act(() => {
            scrollHandler();
        });

        expect(mockLoadMore).toHaveBeenCalled();
    });

    it("should log error when scrollContainerId is not found", () => {
        const consoleSpy = jest.spyOn(console, "error").mockImplementation();
        document.getElementById = jest.fn().mockReturnValue(null);

        renderHook(() => useInfiniteScroll({
            loading: false,
            loadMore: mockLoadMore,
            scrollContainerId: "non-existent-container",
        }));

        expect(consoleSpy).toHaveBeenCalledWith(expect.any(String));

        consoleSpy.mockRestore();
    });
});
