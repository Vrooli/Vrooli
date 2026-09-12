import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { FilterBar } from "@vrooli/react-component-library/FilterBar/1.3.0";
afterEach(cleanup);
it("applies immediate query edits and resets query and filters in one coherent update", () => {
 const onApply = vi.fn();
 render(<FilterBar presentation="inline" applyMode="immediate" onApply={onApply} defaultActiveFilterIds={["open"]} />);
 fireEvent.change(screen.getByRole("searchbox"), { target: { value: "incident" } });
 expect(onApply).toHaveBeenLastCalledWith({query:"incident",activeFilterIds:["open"]});
 onApply.mockClear();
 fireEvent.click(screen.getByRole("button",{name:"Reset"}));
 expect(onApply).toHaveBeenCalledTimes(1);
 expect(onApply).toHaveBeenCalledWith({query:"",activeFilterIds:[]});
 expect(screen.queryByRole("button",{name:"Apply filters"})).toBeNull();
 expect(screen.queryByRole("button",{name:"Reset"})).toBeNull();
});
it("retains explicit submit behavior by default", () => {
 const onApply=vi.fn();render(<FilterBar onApply={onApply} />);
 fireEvent.change(screen.getByRole("searchbox"),{target:{value:"incident"}});
 expect(onApply).not.toHaveBeenCalled();
 fireEvent.click(screen.getByRole("button",{name:"Apply filters"}));
 expect(onApply).toHaveBeenCalledWith({query:"incident",activeFilterIds:[]});
});
