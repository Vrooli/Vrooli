import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";

import { SearchInput } from "./SearchInput";

const ARIA = "search-x";
const CLEAR = "clear-x";

function Controlled({ initial = "", onClear }: { initial?: string; onClear?: () => void }) {
  const [v, setV] = useState(initial);
  return (
    <SearchInput
      value={v}
      onChange={setV}
      ariaLabel={ARIA}
      clearLabel={CLEAR}
      onClear={onClear}
      data-testid="si"
    />
  );
}

describe("SearchInput", () => {
  it("forwards typing through onChange", async () => {
    render(<Controlled />);
    const input = screen.getByRole<HTMLInputElement>("searchbox", { name: ARIA });
    await userEvent.type(input, "abc");
    expect(input.value).toBe("abc");
  });

  it("renders clear button only when value is non-empty and clears on click", async () => {
    const onClear = vi.fn();
    render(<Controlled initial="seed" onClear={onClear} />);
    const clear = screen.getByRole("button", { name: CLEAR });
    await userEvent.click(clear);
    expect(onClear).toHaveBeenCalled();
    const input = screen.getByRole<HTMLInputElement>("searchbox", { name: ARIA });
    expect(input.value).toBe("");
  });
});
