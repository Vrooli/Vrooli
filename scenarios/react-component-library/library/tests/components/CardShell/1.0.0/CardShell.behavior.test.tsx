import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { CardShell } from "../../../../components/CardShell/versions/1.0.0/CardShell";
describe("CardShell",()=>{
 it("shows the selection rail only in selection mode",()=>{const{rerender}=render(<CardShell selection={{selectionMode:false,selected:false}}>content</CardShell>);expect(screen.queryByLabelText("Select row")).toBeNull();rerender(<CardShell selection={{selectionMode:true,selected:false}}>content</CardShell>);expect(screen.getByLabelText("Select row")).toBeTruthy()});
 it("renders disabled reasons and keeps cursor state independent",()=>{render(<CardShell isCursor selection={{selectionMode:true,selected:false,disabled:true,disabledReason:"Limit reached"}}>content</CardShell>);const shell=screen.getByTestId("data-display.card-shell");expect(shell).toHaveAttribute("aria-disabled","true");expect(screen.getByText("Limit reached")).toBeTruthy();expect(shell).toHaveAttribute("data-cursor","true")});
 it("presses selection without invoking row open",()=>{const toggle=vi.fn(),press=vi.fn();render(<CardShell selection={{selectionMode:true,selected:false,onToggleSelect:toggle}} onPress={press}>content</CardShell>);fireEvent.keyDown(screen.getByTestId("data-display.card-shell"),{key:" "});expect(toggle).toHaveBeenCalledOnce();expect(press).not.toHaveBeenCalled()});
});
