import { createServer } from 'node:http';
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { after, before, test } from 'node:test';
import assert from 'node:assert/strict';
import { buildIdentity, sha256File } from './qualification-support.mjs';

let server;
let origin;
let temp;

before(async () => {
  temp = await mkdtemp(join(tmpdir(), 'bas-qualification-support-'));
  server = createServer((request, response) => {
    if (request.url === '/health') {
      response.writeHead(200, { 'content-type': 'application/json' });
      response.end(JSON.stringify({ build_identity: 'sha256:fixture' }));
      return;
    }
    response.writeHead(404).end();
  });
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  origin = `http://127.0.0.1:${server.address().port}`;
});

after(async () => {
  await new Promise((resolve) => server.close(resolve));
  await rm(temp, { recursive: true, force: true });
});

test('normalizes root, api, and trailing-slash health bases', async () => {
  assert.equal(await buildIdentity(origin), 'sha256:fixture');
  assert.equal(await buildIdentity(`${origin}/api/v1`), 'sha256:fixture');
  assert.equal(await buildIdentity(`${origin}/api/v1/`), 'sha256:fixture');
});

test('hashes the exact file bytes', async () => {
  const path = join(temp, 'source.txt');
  await writeFile(path, 'BAS qualification bytes\n');
  assert.equal(await sha256File(path), 'ffc2ed866f231dd8dbd54d6413f8235a39bb5b231461aa66e971b6a33529bb85');
  assert.equal(await readFile(path, 'utf8'), 'BAS qualification bytes\n');
});
