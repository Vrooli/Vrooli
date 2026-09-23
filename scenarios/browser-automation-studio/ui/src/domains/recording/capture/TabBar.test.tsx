import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, cleanup, fireEvent, render } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TabBar } from './TabBar';

const preview = vi.hoisted(() => vi.fn(async () => ({found: true, favicon: 'https://wrong.test/icon.ico'})));
vi.mock('@/api/ai', () => ({aiClient: {getLinkPreview: preview}}));
const page = {id: 'page', sessionId: 'session', url: 'https://fixture.test/one', title: 'One',
  faviconUrl: 'https://fixture.test/custom.svg', createdAt: '2026-09-23T00:00:00Z',
  isInitial: true, status: 'active' as const};
const props = {pages: [page], activePageId: page.id, onTabClick: vi.fn()};

describe('passive browser tab icons [REQ:BAS-RH-J04]', () => {
  beforeEach(() => preview.mockClear());
  afterEach(cleanup);
  it('displays browser-provided custom metadata without fetching the document', async () => {
    const view = render(<TabBar {...props}/>);
    await act(async () => {});
    expect(preview).not.toHaveBeenCalled();
    expect(view.container.querySelector('img')?.getAttribute('src')).toBe(page.faviconUrl);
  });
  it('recovers from a failed old icon when the page supplies a new icon', async () => {
    const view = render(<TabBar {...props}/>);
    await act(async () => {});
    fireEvent.error(view.container.querySelector('img')!);
    expect(view.container.querySelector('img')).toBeNull();
    view.rerender(<TabBar {...props} pages={[{...page, url: 'https://fixture.test/two', faviconUrl: 'https://fixture.test/new.svg'}]}/>);
    await act(async () => {});
    expect(view.container.querySelector('img')?.getAttribute('src')).toBe('https://fixture.test/new.svg');
    expect(preview).not.toHaveBeenCalled();
  });
  it('clears a previous icon when the browser reports none', async () => {
    const view = render(<TabBar {...props}/>);
    await act(async () => {});
    view.rerender(<TabBar {...props} pages={[{...page, faviconUrl: ''}]}/>);
    await act(async () => {});
    expect(view.container.querySelector('img')).toBeNull();
    expect(view.getByRole('tab').textContent).toContain('F');
    expect(preview).not.toHaveBeenCalled();
  });
});

describe('browser tab keyboard controls [REQ:BAS-RH-J13]', () => {
  const pages=[page,{...page,id:'two',title:'Two'},{...page,id:'three',title:'Three'}];
  afterEach(cleanup);

  it('provides one native tab entry and separate close buttons',()=>{
    const view=render(<TabBar pages={pages} activePageId="two" onTabClick={vi.fn()} onTabClose={vi.fn()}/>);
    const tabs=view.getAllByRole('tab');
    expect(tabs.map(t=>t.tagName)).toEqual(['BUTTON','BUTTON','BUTTON']);
    expect(tabs.map(t=>t.tabIndex)).toEqual([-1,0,-1]);
    expect(tabs.every(t=>t.querySelector('button')===null)).toBe(true);
    expect(tabs[1]).toHaveAccessibleName('Two');
  });
  it('wraps arrow focus and supports Home/End without activating remote pages',async()=>{
    const user=userEvent.setup();const select=vi.fn();
    const view=render(<TabBar pages={pages} activePageId="two" onTabClick={select}/>);
    const tabs=view.getAllByRole('tab');act(()=>tabs[1]!.focus());
    await user.keyboard('{ArrowRight}');expect(tabs[2]).toHaveFocus();
    await user.keyboard('{ArrowRight}');expect(tabs[0]).toHaveFocus();
    await user.keyboard('{ArrowLeft}');expect(tabs[2]).toHaveFocus();
    await user.keyboard('{Home}');expect(tabs[0]).toHaveFocus();
    await user.keyboard('{End}');expect(tabs[2]).toHaveFocus();
    expect(select).not.toHaveBeenCalled();
    await user.keyboard('{Enter}');expect(select).toHaveBeenCalledExactlyOnceWith('three');
    select.mockClear();await user.keyboard(' ');expect(select).toHaveBeenCalledExactlyOnceWith('three');
  });
  it('leaves the list with Tab and returns to the selected tab',async()=>{
    const user=userEvent.setup();
    const view=render(<><button>Before</button><TabBar pages={pages} activePageId="two" onTabClick={vi.fn()} onTabClose={vi.fn()}/><button>After</button></>);
    act(()=>view.getByRole('button',{name:'Before'}).focus());
    await user.tab();expect(view.getAllByRole('tab')[1]).toHaveFocus();
    await user.keyboard('{ArrowLeft}');expect(view.getAllByRole('tab')[0]).toHaveFocus();
    await user.tab();expect(view.getByRole('button',{name:'After'})).toHaveFocus();
    await user.tab({shift:true});expect(view.getAllByRole('tab')[1]).toHaveFocus();
  });
  it('closes with Delete and focuses the following tab only after it disappears',async()=>{
    const user=userEvent.setup();const close=vi.fn(),select=vi.fn();
    const p={pages,activePageId:'two',onTabClick:select,onTabClose:close};
    const view=render(<TabBar {...p}/>);const selected=view.getAllByRole('tab')[1]!;
    act(()=>selected.focus());await user.keyboard('{Delete}');
    expect(close).toHaveBeenCalledExactlyOnceWith('two');expect(selected).toHaveFocus();
    expect(select).not.toHaveBeenCalled();
    view.rerender(<TabBar {...p} pages={[pages[0]!,pages[2]!]} activePageId="three"/>);
    expect(view.getAllByRole('tab')[1]).toHaveFocus();
    expect(select).not.toHaveBeenCalled();
  });
  it('focuses the preceding tab at the end and the new-tab control when empty',async()=>{
    const user=userEvent.setup();const p={onTabClick:vi.fn(),onTabClose:vi.fn(),onCreateTab:vi.fn()};
    const view=render(<TabBar {...p} pages={pages} activePageId="three"/>);
    act(()=>view.getAllByRole('tab')[2]!.focus());await user.keyboard('{Delete}');
    view.rerender(<TabBar {...p} pages={[pages[0]!]} activePageId="page"/>);
    expect(view.getByRole('tab')).toHaveFocus();
    await user.keyboard('{Delete}');view.rerender(<TabBar {...p} pages={[]} activePageId={null}/>);
    expect(view.getByRole('tab')).toHaveFocus();await user.keyboard(' ');
    expect(p.onCreateTab).toHaveBeenCalledTimes(1);
  });
  it('does not steal focus when a tab closes after the user leaves the bar',async()=>{
    const user=userEvent.setup();const p={onTabClick:vi.fn(),onTabClose:vi.fn()};
    const view=render(<><TabBar {...p} pages={pages} activePageId="two"/><button>Outside</button></>);
    act(()=>view.getAllByRole('tab')[1]!.focus());await user.keyboard('{Delete}');
    await user.click(view.getByRole('button',{name:'Outside'}));
    view.rerender(<><TabBar {...p} pages={[pages[0]!]} activePageId="page"/><button>Outside</button></>);
    expect(view.getByRole('button',{name:'Outside'})).toHaveFocus();
  });
  it('retains pointer close without selecting the closed page',async()=>{
    const user=userEvent.setup();const close=vi.fn(),select=vi.fn();
    const view=render(<TabBar pages={pages} activePageId="two" onTabClick={select} onTabClose={close}/>);
    await user.click(view.getByRole('button',{name:'Close One'}));
    expect(close).toHaveBeenCalledExactlyOnceWith('page');expect(select).not.toHaveBeenCalled();
  });
  it('keeps an entry while selection is pending and disables unavailable creation',()=>{
    const view=render(<TabBar pages={pages} activePageId={null} onTabClick={vi.fn()}/>);
    expect(view.getAllByRole('tab').map(t=>t.tabIndex)).toEqual([0,-1,-1]);
    view.rerender(<TabBar pages={[]} activePageId={null} onTabClick={vi.fn()}/>);
    expect(view.getByRole('tab')).toBeDisabled();
  });
  it('keeps focus when the empty control is replaced by the created tab',async()=>{
    const user=userEvent.setup();const p={onTabClick:vi.fn(),onCreateTab:vi.fn()};
    const view=render(<TabBar {...p} pages={[]} activePageId={null}/>);
    act(()=>view.getByRole('tab').focus());await user.keyboard('{Enter}');
    expect(p.onCreateTab).toHaveBeenCalledTimes(1);expect(view.getByRole('tab')).toHaveFocus();
    view.rerender(<TabBar {...p} pages={[page]} activePageId={page.id}/>);
    expect(view.getByRole('tab')).toHaveFocus();
  });
});
