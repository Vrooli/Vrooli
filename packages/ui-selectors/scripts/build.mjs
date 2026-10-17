import { cpSync, mkdirSync } from 'node:fs';
mkdirSync(new URL('../dist/', import.meta.url), { recursive: true });
cpSync(new URL('../src/', import.meta.url), new URL('../dist/', import.meta.url), { recursive: true });
