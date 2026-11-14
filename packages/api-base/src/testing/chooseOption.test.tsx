import { cleanup, render, screen } from "@testing-library/react";
import { useId, useState } from "react";
import { afterEach, describe, expect, it } from "vitest";

import { chooseOption } from "./index.js";

const options = [
  { value: "oven", label: "Oven" },
  { value: "grill", label: "Grill" },
];

// The shape a React Component Library Select renders: a hidden native select
// for form state, and a trigger named by the consumer's <label for>.
function LibrarySelect({ id, onChange }: { id: string; onChange: (value: string) => void }) {
  const [value, setValue] = useState("oven");
  const [open, setOpen] = useState(false);
  const labelId = `${id}-label`;
  const choose = (next: string) => {
    setValue(next);
    setOpen(false);
    onChange(next);
  };
  return (
    <span>
      <select id={id} aria-hidden="true" tabIndex={-1} value={value} onChange={(event) => choose(event.target.value)}>
        {options.map((option) => (
          <option key={option.value} value={option.value}>{option.label}</option>
        ))}
      </select>
      <button type="button" aria-labelledby={labelId} aria-haspopup="listbox" onClick={() => setOpen(true)}>
        {options.find((option) => option.value === value)?.label}
      </button>
      {open ? (
        <div role="listbox">
          {options.map((option) => (
            <button key={option.value} type="button" role="option" aria-selected={option.value === value} onClick={() => choose(option.value)}>
              {option.label}
            </button>
          ))}
        </div>
      ) : null}
    </span>
  );
}

function Field({ library, onChange }: { library: boolean; onChange: (value: string) => void }) {
  const id = useId();
  return (
    <>
      <label id={`${id}-label`} htmlFor={id}>Method</label>
      {library ? (
        <LibrarySelect id={id} onChange={onChange} />
      ) : (
        <select id={id} defaultValue="oven" onChange={(event) => onChange(event.target.value)}>
          {options.map((option) => (
            <option key={option.value} value={option.value}>{option.label}</option>
          ))}
        </select>
      )}
    </>
  );
}

afterEach(cleanup);

describe("chooseOption", () => {
  it.each([true, false])("chooses by label or by value (library select: %s)", async (library) => {
    const chosen: string[] = [];
    render(<Field library={library} onChange={(value) => chosen.push(value)} />);

    await chooseOption("Method", "Grill");
    await chooseOption("Method", "oven");

    expect(chosen).toEqual(["grill", "oven"]);
  });

  it("reports an option the native select does not offer", async () => {
    render(<Field library={false} onChange={() => undefined} />);
    await expect(chooseOption("Method", "Fryer")).rejects.toThrow(/no option "Fryer"/);
    expect(screen.getByLabelText("Method")).toHaveProperty("value", "oven");
  });
});
