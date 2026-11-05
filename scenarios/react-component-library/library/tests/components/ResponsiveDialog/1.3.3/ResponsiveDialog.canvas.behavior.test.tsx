import { describe, expect, it } from 'vitest';
import { responsiveDialogStyles } from '@vrooli/react-component-library/ResponsiveDialog/1.3.3';
// ResponsiveDialog is the Commit/Push presentation. It must have the same
// bottom canvas ownership as FullPageDrawer, independently of scenario CSS.
describe('responsive mobile sheet canvas',()=>{
 it('colors both page canvas layers only while the sheet is open',()=>{
  expect(responsiveDialogStyles).toContain('html:has([data-rcl-responsive-dialog][data-state="open"]) body { background-color: var(--color-surface-raised); }');
 });
});
