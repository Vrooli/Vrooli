import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { Pagination } from "./pagination";

describe("Pagination", () => {
  it("renders nothing when total fits in one page", () => {
    const { container } = render(
      <Pagination page={0} total={5} pageSize={10} onPageChange={vi.fn()} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders page info and controls when multiple pages exist", () => {
    render(<Pagination page={0} total={50} pageSize={10} onPageChange={vi.fn()} />);
    expect(screen.getByText("1–10 of 50")).toBeInTheDocument();
    expect(screen.getByText("1 / 5")).toBeInTheDocument();
  });

  it("disables prev button on first page", () => {
    render(<Pagination page={0} total={30} pageSize={10} onPageChange={vi.fn()} />);
    expect(screen.getByLabelText("Previous page")).toBeDisabled();
    expect(screen.getByLabelText("Next page")).not.toBeDisabled();
  });

  it("disables next button on last page", () => {
    render(<Pagination page={2} total={30} pageSize={10} onPageChange={vi.fn()} />);
    expect(screen.getByLabelText("Next page")).toBeDisabled();
    expect(screen.getByLabelText("Previous page")).not.toBeDisabled();
  });

  it("calls onPageChange with correct page on prev click", () => {
    const onChange = vi.fn();
    render(<Pagination page={2} total={50} pageSize={10} onPageChange={onChange} />);
    fireEvent.click(screen.getByLabelText("Previous page"));
    expect(onChange).toHaveBeenCalledWith(1);
  });

  it("calls onPageChange with correct page on next click", () => {
    const onChange = vi.fn();
    render(<Pagination page={1} total={50} pageSize={10} onPageChange={onChange} />);
    fireEvent.click(screen.getByLabelText("Next page"));
    expect(onChange).toHaveBeenCalledWith(2);
  });

  it("shows correct range for last partial page", () => {
    render(<Pagination page={2} total={25} pageSize={10} onPageChange={vi.fn()} />);
    expect(screen.getByText("21–25 of 25")).toBeInTheDocument();
    expect(screen.getByText("3 / 3")).toBeInTheDocument();
  });

  it("has data-testid attributes for selectors", () => {
    render(<Pagination page={0} total={20} pageSize={10} onPageChange={vi.fn()} />);
    expect(screen.getByTestId("pagination")).toBeInTheDocument();
    expect(screen.getByTestId("pagination-prev")).toBeInTheDocument();
    expect(screen.getByTestId("pagination-next")).toBeInTheDocument();
  });
});
