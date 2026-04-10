import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { showActivationOverlay } from '../activationOverlay.js';

describe('showActivationOverlay', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    document.body.innerHTML = '';
    document.head.innerHTML = '';
  });

  afterEach(() => {
    document.body.innerHTML = '';
    document.head.innerHTML = '';
    vi.useRealTimers();
  });

  describe('when: true (forced, no fade-in delay)', () => {
    it('creates a visible overlay immediately', () => {
      const handle = showActivationOverlay({ when: true });

      expect(handle.element).toBeInstanceOf(HTMLElement);
      expect(handle.shown).toBe(true);
      expect(handle.element!.hasAttribute('data-visible')).toBe(true);
      expect(handle.element!.hasAttribute('data-auto')).toBe(false);
      expect(document.body.contains(handle.element)).toBe(true);

      handle.dismiss();
      vi.advanceTimersByTime(500);
    });

    it('injects styles into the head', () => {
      const handle = showActivationOverlay({ when: true });

      const style = document.querySelector('style[data-spatial-activation-styles]');
      expect(style).not.toBeNull();

      handle.dismiss();
      vi.advanceTimersByTime(500);
    });

    it('displays custom message and sub-message', () => {
      const handle = showActivationOverlay({
        when: true,
        message: 'Custom message',
        subMessage: 'Custom sub',
      });

      expect(handle.element!.textContent).toContain('Custom message');
      expect(handle.element!.textContent).toContain('Custom sub');

      handle.dismiss();
      vi.advanceTimersByTime(500);
    });

    it('displays default messages', () => {
      const handle = showActivationOverlay({ when: true });

      expect(handle.element!.textContent).toContain('Press any button on your controller');
      expect(handle.element!.textContent).toContain('Tap anywhere to dismiss');

      handle.dismiss();
      vi.advanceTimersByTime(500);
    });

    it('dismiss() removes overlay from DOM', () => {
      const handle = showActivationOverlay({ when: true });
      expect(document.body.contains(handle.element)).toBe(true);

      handle.dismiss();
      vi.advanceTimersByTime(500);

      expect(document.body.contains(handle.element)).toBe(false);
    });
  });

  describe('when: auto (default — delayed fade-in)', () => {
    it('starts invisible with data-auto attribute', () => {
      const handle = showActivationOverlay({ when: 'auto' });

      expect(handle.shown).toBe(true);
      expect(handle.element!.hasAttribute('data-auto')).toBe(true);
      expect(handle.element!.hasAttribute('data-visible')).toBe(false);

      handle.dismiss();
      vi.advanceTimersByTime(500);
    });

    it('becomes visible after fade-in delay', () => {
      const handle = showActivationOverlay({ when: 'auto' });

      // Before delay — still invisible
      vi.advanceTimersByTime(300);
      expect(handle.element!.hasAttribute('data-auto')).toBe(true);

      // After delay — becomes visible
      vi.advanceTimersByTime(200);
      expect(handle.element!.hasAttribute('data-visible')).toBe(true);
      expect(handle.element!.hasAttribute('data-auto')).toBe(false);

      handle.dismiss();
      vi.advanceTimersByTime(500);
    });

    it('never becomes visible if dismissed before delay', () => {
      const handle = showActivationOverlay({ when: 'auto' });

      // Dismiss via mouse before the fade-in delay
      vi.advanceTimersByTime(100);
      window.dispatchEvent(new MouseEvent('mousemove'));
      vi.advanceTimersByTime(500);

      // It was dismissed before becoming visible
      expect(document.body.contains(handle.element)).toBe(false);
    });

    it('defaults to auto when when is omitted', () => {
      const handle = showActivationOverlay();

      expect(handle.shown).toBe(true);
      expect(handle.element!.hasAttribute('data-auto')).toBe(true);

      handle.dismiss();
      vi.advanceTimersByTime(500);
    });
  });

  describe('when: false', () => {
    it('does not create an overlay', () => {
      const handle = showActivationOverlay({ when: false });

      expect(handle.shown).toBe(false);
      expect(handle.element).toBeNull();
      expect(document.querySelector('[data-spatial-activation]')).toBeNull();
    });
  });

  describe('dismiss triggers', () => {
    it('auto-dismisses on gamepadconnected', () => {
      const onDismiss = vi.fn();
      const handle = showActivationOverlay({ when: true, onDismiss });

      window.dispatchEvent(new Event('gamepadconnected'));
      vi.advanceTimersByTime(500);

      expect(onDismiss).toHaveBeenCalledWith('gamepad');
      expect(document.body.contains(handle.element)).toBe(false);
    });

    it('auto-dismisses on mousemove', () => {
      const onDismiss = vi.fn();
      showActivationOverlay({ when: true, onDismiss });

      window.dispatchEvent(new MouseEvent('mousemove'));
      vi.advanceTimersByTime(500);

      expect(onDismiss).toHaveBeenCalledWith('mouse');
    });

    it('auto-dismisses on keydown', () => {
      const onDismiss = vi.fn();
      showActivationOverlay({ when: true, onDismiss });

      window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
      vi.advanceTimersByTime(500);

      expect(onDismiss).toHaveBeenCalledWith('keyboard');
    });

    it('auto-dismisses on touchstart', () => {
      const onDismiss = vi.fn();
      showActivationOverlay({ when: true, onDismiss });

      window.dispatchEvent(new Event('touchstart'));
      vi.advanceTimersByTime(500);

      expect(onDismiss).toHaveBeenCalledWith('touch');
    });

    it('auto-dismisses on overlay click (tap)', () => {
      const onDismiss = vi.fn();
      const handle = showActivationOverlay({ when: true, onDismiss });

      handle.element!.click();
      vi.advanceTimersByTime(500);

      expect(onDismiss).toHaveBeenCalledWith('touch');
    });

    it('only dismisses once on multiple events', () => {
      const onDismiss = vi.fn();
      showActivationOverlay({ when: true, onDismiss });

      window.dispatchEvent(new Event('gamepadconnected'));
      window.dispatchEvent(new MouseEvent('mousemove'));
      window.dispatchEvent(new KeyboardEvent('keydown'));
      vi.advanceTimersByTime(500);

      expect(onDismiss).toHaveBeenCalledTimes(1);
      expect(onDismiss).toHaveBeenCalledWith('gamepad');
    });
  });

  it('does not duplicate styles on multiple calls', () => {
    const h1 = showActivationOverlay({ when: true });
    h1.dismiss();
    vi.advanceTimersByTime(500);

    const h2 = showActivationOverlay({ when: true });

    const styles = document.querySelectorAll('style[data-spatial-activation-styles]');
    expect(styles.length).toBe(1);

    h2.dismiss();
    vi.advanceTimersByTime(500);
  });
});
