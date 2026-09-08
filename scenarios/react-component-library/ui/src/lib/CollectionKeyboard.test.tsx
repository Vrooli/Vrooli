import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { useCollection } from "@vrooli/react-component-library/useCollection/1.1.0";
afterEach(cleanup);
function Fixture({onRevealRow}:{onRevealRow?:(key:string)=>void}) {
 const items=[{id:"a"},{id:"b"},{id:"c"}];
 const collection=useCollection(items,{getKey:item=>item.id,onRevealRow});
 return <div {...collection.getContainerProps()}>{items.map(item=><div key={item.id} {...collection.getRowProps(item)}>{item.id}<input aria-label={`Edit ${item.id}`} /></div>)}</div>;
}
it("moves real focus with the keyboard cursor and reveals the destination",()=>{
 const reveal=vi.fn();render(<Fixture onRevealRow={reveal} />);
 const rows=screen.getAllByRole("listitem");act(()=>rows[0].focus());fireEvent.keyDown(rows[0],{key:"ArrowDown"});
 expect(rows[1]).toHaveFocus();expect(rows[1]).toHaveAttribute("tabindex","0");expect(reveal).toHaveBeenCalledWith("b");
 fireEvent.keyDown(rows[1],{key:"ArrowUp"});expect(rows[0]).toHaveFocus();
});
it("does not steal arrow keys from an embedded editor",()=>{
 const reveal=vi.fn();render(<Fixture onRevealRow={reveal} />);
 const editor=screen.getByRole("textbox",{name:"Edit a"});act(()=>editor.focus());fireEvent.keyDown(editor,{key:"ArrowDown"});
 expect(editor).toHaveFocus();expect(reveal).not.toHaveBeenCalled();
});
