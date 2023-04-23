import { build } from 'vite';

(async () => {
    await build({
        configFile: './vite.config.server.ts',
    });
})();