import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { Popover, PopoverTrigger, PopoverContent } from "./popover";

describe("Popover", () => {
  it("renders trigger but not content initially", () => {
    render(
      <Popover>
        <PopoverTrigger>
          <button>Open</button>
        </PopoverTrigger>
        <PopoverContent>
          <div>Content</div>
        </PopoverContent>
      </Popover>,
    );

    expect(screen.getByText("Open")).toBeInTheDocument();
    expect(screen.queryByText("Content")).not.toBeInTheDocument();
  });

  it("opens on trigger click", () => {
    render(
      <Popover>
        <PopoverTrigger>
          <button>Open</button>
        </PopoverTrigger>
        <PopoverContent>
          <div>Content</div>
        </PopoverContent>
      </Popover>,
    );

    fireEvent.click(screen.getByText("Open"));
    expect(screen.getByText("Content")).toBeInTheDocument();
  });

  it("closes on outside click", () => {
    render(
      <div>
        <div data-testid="outside">Outside</div>
        <Popover>
          <PopoverTrigger>
            <button>Open</button>
          </PopoverTrigger>
          <PopoverContent>
            <div>Content</div>
          </PopoverContent>
        </Popover>
      </div>,
    );

    fireEvent.click(screen.getByText("Open"));
    expect(screen.getByText("Content")).toBeInTheDocument();

    fireEvent.mouseDown(screen.getByTestId("outside"));
    expect(screen.queryByText("Content")).not.toBeInTheDocument();
  });

  it("closes on Escape key", () => {
    render(
      <Popover>
        <PopoverTrigger>
          <button>Open</button>
        </PopoverTrigger>
        <PopoverContent>
          <div>Content</div>
        </PopoverContent>
      </Popover>,
    );

    fireEvent.click(screen.getByText("Open"));
    expect(screen.getByText("Content")).toBeInTheDocument();

    fireEvent.keyDown(document, { key: "Escape" });
    expect(screen.queryByText("Content")).not.toBeInTheDocument();
  });

  it("renders content with dialog role", () => {
    render(
      <Popover>
        <PopoverTrigger>
          <button>Open</button>
        </PopoverTrigger>
        <PopoverContent>
          <div>Content</div>
        </PopoverContent>
      </Popover>,
    );

    fireEvent.click(screen.getByText("Open"));
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });
});
