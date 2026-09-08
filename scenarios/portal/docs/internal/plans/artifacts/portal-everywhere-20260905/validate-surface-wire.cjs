// Reproduce with node <this-file> from the Vrooli repository root.
// Uses Portal's existing governed TypeScript/protobuf tools; installs nothing.
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const assert = require('node:assert/strict');
const { createRequire } = require('node:module');
const root = process.cwd();
const requireUI = createRequire(path.join(root, 'scenarios/portal/ui/package.json'));
const ts = requireUI('typescript');
const { fromJson, toJson, toBinary, fromBinary } = requireUI('@bufbuild/protobuf');
const temporary = fs.mkdtempSync(path.join(os.tmpdir(), 'portal-surface-wire-'));
try {
  fs.symlinkSync(path.join(root, 'scenarios/portal/ui/node_modules'), path.join(temporary, 'node_modules'), 'dir');
  for (const relative of ['buf/validate/validate_pb', 'common/v1/surface_pb']) {
    const source = fs.readFileSync(path.join(root, 'packages/proto/gen/typescript', relative + '.ts'), 'utf8');
    const result = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.CommonJS } });
    const output = path.join(temporary, relative + '.js');
    fs.mkdirSync(path.dirname(output), { recursive: true });
    fs.writeFileSync(output, result.outputText);
  }
  const { SurfaceDescriptorSchema } = require(path.join(temporary, 'common/v1/surface_pb.js'));
  const fixture = JSON.parse(fs.readFileSync(path.join(root, 'packages/api-core/targetmodel/testdata/surface.json'), 'utf8'));
  const message = fromJson(SurfaceDescriptorSchema, fixture);
  const roundtrip = fromBinary(SurfaceDescriptorSchema, toBinary(SurfaceDescriptorSchema, message));
  assert.deepEqual(toJson(SurfaceDescriptorSchema, roundtrip, { useProtoFieldName: true }), fixture);
  assert.equal(roundtrip.ref.target.hostNodeId, 'node-1');
  assert.equal(roundtrip.desktopSessionId, 'login-2');
  assert.throws(() => fromJson(SurfaceDescriptorSchema, { ...fixture, token: 'unauthorized' }));
  assert.throws(() => fromJson(SurfaceDescriptorSchema, { ...fixture, kind: 'SHELL' }));
  console.log('PASS: TypeScript JSON/binary round trip preserves the shared Go fixture and rejects unknown authority fields and enum names.');
} finally {
  fs.rmSync(temporary, { recursive: true, force: true });
}
