import { action, env, runAction, withLeases } from './support';

describe('[REQ:BAS-RH-J04] navigation and browser document behaviors', () => {
  it('given redirect, SPA, popup, shadow DOM and service worker fixtures, when driven, then each surface remains addressable', () => withLeases(async (open) => {
    const lease = await open();
    const redirected = await runAction(lease, action('navigate', { url: `${env.fixture}/redirect` }));
    expect(redirected.finalUrl).toBe(`${env.fixture}/redirect-target`);

    await runAction(lease, action('navigate', { url: `${env.fixture}/complex` }));
    const spa = await runAction(lease, action('click', { selector: '#spa' }));
    expect(spa.finalUrl).toBe(`${env.fixture}/spa/next`);
    await runAction(lease, action('click', { selector: '#shadow-host >>> #shadow-button' }));
    await runAction(lease, action('click', { selector: '#popup' }));
    await runAction(lease, action('tab-switch', { action: 'switch', urlPattern: '/same-popup' }));
    expect((await runAction(lease, action('click', { selector: '#same' }))).success).toBe(true);
    await runAction(lease, action('tab-switch', { action: 'switch', index: 0 }));
    await runAction(lease, action('navigate', { url: `${env.fixture}/service-worker`, waitForSelector: '#worker-result' }));
    await runAction(lease, action('wait', { selector: '#worker-result.worker-ok' }));
  }));
});
