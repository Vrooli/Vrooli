/** Transparent Web Audio constructor patch for hosts without an output device. */
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
