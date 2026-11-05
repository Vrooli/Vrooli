/**
 * Transparent Web Audio constructor patch for hosts without an output device.
 *
 * Measured effect, interleaved A/B over three repetitions on a host whose every
 * audio output profile reports itself unavailable:
 *
 * - Capture-driven graph (a `getUserMedia` stream from
 *   `--use-file-for-fake-audio-capture`): 32.3 s wall and clock rate 0.062
 *   unpatched, versus 2.3 s and 0.996 patched. This is the path deterministic
 *   audio workflows use, and the reason this patch exists.
 * - Output-only graph (an oscillator with no capture stream): unchanged by the
 *   patch. Both patched and unpatched stall about 30 s and then advance at
 *   roughly 0.25x. The silent sink removes the blocking output device; it does
 *   not supply a clock, and an output-only graph has no other clock source.
 *
 * Do not treat a flat reading from an output-only probe as evidence that this
 * patch is inactive. `detectHostAudioCapability` deliberately uses an
 * output-only context, because a stalled clock there is exactly the signal that
 * the host has no usable output device.
 */
export function generateSilentSinkPatch(): string {
  return `
    (() => {
      const patch = (name) => {
        const Native = window[name];
        if (!Native) return;
        function SilentSinkAudioContext(options) {
          const merged = { ...(options || {}), sinkId: { type: 'none' } };
          let context;
          try { context = new Native(merged); } catch (_) { context = new Native(options); }
          // Some Chromium versions accept a string sinkId in the constructor
          // but only apply AudioSinkOptions through the stable async method.
          // Request it in both forms; failures preserve the native context.
          if (typeof context.setSinkId === 'function') {
            void context.setSinkId({ type: 'none' }).catch(() => undefined);
          }
          return context;
        }
        SilentSinkAudioContext.prototype = Native.prototype;
        Object.setPrototypeOf(SilentSinkAudioContext, Native);
        window[name] = SilentSinkAudioContext;
      };
      patch('AudioContext'); patch('webkitAudioContext');
    })();
  `;
}
