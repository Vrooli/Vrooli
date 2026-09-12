import { useRef, useState, type ReactNode } from "react";
import { Loader2, MoreVertical } from "lucide-react";
import { ContextMenu, type ContextMenuItem } from "@vrooli/react-component-library/ContextMenu/1";
import { Button, type ButtonProps } from "./button";
import { cn } from "../../lib/utils";

export interface ActionMenuItem { label: string; description?: string; onSelect: () => void; icon?: ReactNode; disabled?: boolean; destructive?: boolean; loading?: boolean; title?: string; testId?: string }
export interface ActionMenuProps { items: ActionMenuItem[]; label?: string; triggerTestId?: string; menuTestId?: string; triggerIcon?: ReactNode; triggerVariant?: ButtonProps["variant"]; triggerSize?: ButtonProps["size"]; onItemSelected?: () => void; className?: string; mobileSheet?: boolean }

function toContextItems(items: ActionMenuItem[], close: () => void): ContextMenuItem[] { return items.map((item, index) => ({ id: `${index}-${item.label}`, label: item.label, icon: item.loading ? <Loader2 className="animate-spin" /> : item.icon, disabled: item.disabled, destructive: item.destructive, testId: item.testId, onSelect: () => { if (!item.disabled) { item.onSelect(); close(); } } })); }

export function ActionMenu({ items, label = "Actions", triggerTestId, menuTestId, triggerIcon = <MoreVertical className="h-4 w-4" />, triggerVariant = "ghost", triggerSize = "icon", onItemSelected, className }: ActionMenuProps) {
  const [open, setOpen] = useState(false); const triggerRef = useRef<HTMLButtonElement>(null); if (!items.length) return null;
  const close = () => { setOpen(false); onItemSelected?.(); };
  return <><Button ref={triggerRef} variant={triggerVariant} size={triggerSize} aria-label={label} aria-haspopup="menu" aria-expanded={open} title={label} data-testid={triggerTestId} className={className} onClick={() => setOpen((value) => !value)}>{triggerIcon}</Button>{open && <ContextMenu open onOpenChange={setOpen} anchorRef={triggerRef} title={label} closeLabel={`Close ${label.toLocaleLowerCase()}`} testId={menuTestId} triggers={[]} items={toContextItems(items, close)} />}</>;
}
export function ActionMenuSheetContent({ items, onItemSelected, className }: { items: ActionMenuItem[]; onItemSelected?: () => void; className?: string }) { return <ActionMenuItems items={items} onItemSelected={onItemSelected} className={cn("py-1", className)} />; }
export function ActionMenuPanel({ children, className, testId }: { children: ReactNode; className?: string; testId?: string }) { return <div role="menu" className={cn("min-w-[200px] overflow-hidden rounded-md border border-white/10 bg-slate-900 py-1 shadow-lg", className)} data-testid={testId}>{children}</div>; }
export function ActionMenuItems({ items, onItemSelected, className, role, itemRole }: { items: ActionMenuItem[]; onItemSelected?: () => void; className?: string; role?: "menu"; itemRole?: "menuitem" }) { return <div className={cn("flex flex-col", className)} role={role}>{items.map((item, index) => <ActionMenuItemButton key={`${index}-${item.label}`} item={item} onItemSelected={onItemSelected} role={itemRole} />)}</div>; }
export function ActionMenuItemButton({ item, onItemSelected, role }: { item: ActionMenuItem; onItemSelected?: () => void; role?: "menuitem" }) { return <button type="button" role={role} disabled={item.disabled} title={item.title} onClick={(event) => { event.preventDefault(); if (!item.disabled) { onItemSelected?.(); item.onSelect(); } }} className={cn("flex min-h-12 w-full items-start gap-3 px-3 py-2.5 text-left transition-colors disabled:cursor-not-allowed disabled:opacity-50 [&>svg]:mt-0.5 [&>svg]:h-4 [&>svg]:w-4", item.destructive ? "text-red-300 hover:bg-red-500/10" : "text-slate-200 hover:bg-slate-800")} data-testid={item.testId}>{item.loading ? <Loader2 className="animate-spin" /> : item.icon}<span className="min-w-0"><span className="block truncate text-sm font-medium">{item.label}</span>{item.description && <span className="mt-0.5 block text-xs leading-4 text-slate-400">{item.description}</span>}</span></button>; }
export function ActionMenuSeparator() { return <div className="my-1 h-px bg-slate-800" role="separator" />; }
