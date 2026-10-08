import { action, env, runAction, withLeases } from './support';

describe('[REQ:BAS-RH-J03] same selector across tabs and frames', () => {
  it('given identical controls in a frame and a second tab, when each is selected, then the matching document receives the click', () => withLeases(async (open) => {
    const lease = await open();
    await runAction(lease, action('navigate', { url: `${env.fixture}/frames` }));
    await runAction(lease, action('frame-switch', { action: 'enter', selector: '#fixture-frame' }));
    await runAction(lease, action('click', { selector: '#same' }));
    await runAction(lease, action('frame-switch', { action: 'exit' }));
    await runAction(lease, action('tab-switch', { action: 'open', url: `${env.fixture}/same` }));
    await runAction(lease, action('click', { selector: '#same' }));
  }));
});
