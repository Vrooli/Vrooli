import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import MessagesSearchDrawer from "../components/MessagesSearchDrawer";
import { strings } from "../consts/strings";
import { i18n } from "../i18n";

const defaultProps = {
  open: true,
  onClose: vi.fn(),
  query: "",
  onQueryChange: vi.fn(),
  matchCount: 0,
  currentMatchIndex: -1,
  onPrevMatch: vi.fn(),
  onNextMatch: vi.fn(),
};

describe("MessagesSearchDrawer", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders search input, prev/next, and close when open", () => {
    render(<MessagesSearchDrawer {...defaultProps} />);

    expect(screen.getByTestId("messages-search-input")).toBeInTheDocument();
    expect(screen.getByTestId("messages-search-prev")).toBeInTheDocument();
    expect(screen.getByTestId("messages-search-next")).toBeInTheDocument();
    expect(screen.getByTestId("messages-search-close")).toBeInTheDocument();
  });

  it("does not render when open is false", () => {
    render(<MessagesSearchDrawer {...defaultProps} open={false} />);

    expect(screen.queryByTestId("messages-search-input")).toBeNull();
    expect(screen.queryByTestId("messages-search-panel")).toBeNull();
  });

  it("calls onQueryChange when typing in search input", () => {
    render(<MessagesSearchDrawer {...defaultProps} />);

    fireEvent.change(screen.getByTestId("messages-search-input"), {
      target: { value: "hello" },
    });
    expect(defaultProps.onQueryChange).toHaveBeenCalledWith("hello");
  });

  it("displays match count correctly", async () => {
    await i18n.changeLanguage("en");
    render(
      <MessagesSearchDrawer
        {...defaultProps}
        query="test"
        matchCount={5}
        currentMatchIndex={1}
      />,
    );

    expect(screen.getByTestId("messages-search-match-count")).toHaveTextContent("2 of 5");
  });

  it("shows 'No matches' when query exists but matchCount is 0", () => {
    render(
      <MessagesSearchDrawer
        {...defaultProps}
        query="nonexistent"
        matchCount={0}
        currentMatchIndex={-1}
      />,
    );

    expect(screen.getByTestId("messages-search-match-count")).toHaveTextContent(strings.messagesSearch.noMatches);
  });

  it("shows 'Type to search' when query is empty", () => {
    render(<MessagesSearchDrawer {...defaultProps} query="" matchCount={0} />);

    expect(screen.getByTestId("messages-search-match-count")).toHaveTextContent(strings.messagesSearch.typeToSearch);
  });

  it("disables prev/next when query is empty", () => {
    render(<MessagesSearchDrawer {...defaultProps} query="" matchCount={0} />);

    expect(screen.getByTestId("messages-search-prev")).toBeDisabled();
    expect(screen.getByTestId("messages-search-next")).toBeDisabled();
  });

  it("calls onPrevMatch and onNextMatch on button clicks", () => {
    render(
      <MessagesSearchDrawer
        {...defaultProps}
        query="test"
        matchCount={3}
        currentMatchIndex={1}
      />,
    );

    fireEvent.click(screen.getByTestId("messages-search-prev"));
    expect(defaultProps.onPrevMatch).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByTestId("messages-search-next"));
    expect(defaultProps.onNextMatch).toHaveBeenCalledTimes(1);
  });

  it("disables prev/next when matchCount is 0", () => {
    render(
      <MessagesSearchDrawer
        {...defaultProps}
        query="test"
        matchCount={0}
        currentMatchIndex={-1}
      />,
    );

    expect(screen.getByTestId("messages-search-prev")).toBeDisabled();
    expect(screen.getByTestId("messages-search-next")).toBeDisabled();
  });

  it("calls onClose when close button is clicked", () => {
    render(<MessagesSearchDrawer {...defaultProps} />);

    fireEvent.click(screen.getByTestId("messages-search-close"));
    expect(defaultProps.onClose).toHaveBeenCalledTimes(1);
  });

  it("calls onClose when backdrop is clicked", () => {
    render(<MessagesSearchDrawer {...defaultProps} />);

    fireEvent.click(screen.getByTestId("messages-search-backdrop"));
    expect(defaultProps.onClose).toHaveBeenCalledTimes(1);
  });
});
