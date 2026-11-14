import { action, env, fixtureAfter, fixtureInput, fixtureState, runAction, withLeases } from './support';

describe('[REQ:BAS-RH-J13] keyboard shortcuts, drag and scroll', () => {
  it('given a local gesture surface, when shortcut, drag and scroll actions run, then all are accepted by the active page', () => withLeases(async (open) => {
    const lease = await open();
    await runAction(lease, action('navigate', { url: `${env.fixture}/gestures` }));
    const before = await fixtureState();
    await runAction(lease, action('input', { selector: '#shortcut', value: 'replace me' }));
    await runAction(lease, action('shortcut', { selector: '#shortcut', shortcut: 'Control+A' }));
    await runAction(lease, action('keyboard', { key: 'x' }));
    expect((await fixtureInput((input) => input.value === 'x')).value).toBe('x');
    await runAction(lease, action('drag-drop', { source: '#drag-source', target: '#drag-target' }));
    expect((await fixtureAfter('effect', before.effects.length)).effects).toHaveLength(before.effects.length + 1);
    await runAction(lease, action('scroll', { y: 500 }));
    expect((await fixtureAfter('scroll', before.scrolls.length)).scrolls.at(-1)?.y).toBeGreaterThan(0);
  }));
});
