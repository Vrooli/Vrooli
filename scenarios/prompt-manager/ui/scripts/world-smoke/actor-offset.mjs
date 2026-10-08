// Normal workbench edits distinguish animation attribute updates from mesh resolution rebuilds.
import{chromium}from'playwright-core';import{writeFileSync}from'node:fs';import{installFixtureHook}from'./camera-fixtures.mjs';
const root=new URL('../../evidence/living-world/',import.meta.url).pathname;const b=await chromium.launch({executablePath:'/usr/bin/google-chrome',args:['--no-sandbox','--ignore-gpu-blocklist','--use-gl=angle','--use-angle=gl-egl']});const p=await b.newPage({viewport:{width:1400,height:900}});await installFixtureHook(p);const checks=[],errors=[];p.on('pageerror',e=>errors.push(e.message));p.on('console',m=>{if(m.type()==='error'&&/shader|WebGL|THREE/.test(m.text()))errors.push(m.text())});const check=(name,pass,detail)=>{checks.push({name,pass,detail});if(!pass)throw Error(name)};try{await p.goto('http://localhost:21235/world?scene=park&actors=25&seed=1&profile=high&diag=1&workbench=1');await p.waitForFunction(()=>window.__worldDiagnostics?.ready,null,{timeout:90000});await p.evaluate(()=>{const pending=[...window.__cameraFixtureRoots].map(r=>r.current);let render;while(pending.length){const f=pending.pop();for(const v of[f.memoizedProps?.store,f.memoizedProps?.value])if(typeof v?.getState==='function'&&v.getState().scene?.isScene)render=v;if(f.child)pending.push(f.child);if(f.sibling)pending.push(f.sibling)}window.__readSlime=()=>{let mesh;render.getState().scene.traverse(o=>{if(o.material?.slime)mesh=o});return mesh};const mesh=window.__readSlime();window.__slimeBefore={mesh,material:mesh.material,geometry:mesh.geometry,uniforms:mesh.material.slime,version:mesh.material.version,generation:window.__worldSim.generation().count}});await p.getByTitle('World Settings',{exact:true}).click();await p.getByText('World workbench',{exact:true}).click();await p.getByRole('button',{name:'actor',exact:true}).click();
const read = () => p.evaluate(() => {
  const mesh = window.__readSlime(), before = window.__slimeBefore;
  const shifts = mesh.geometry.getAttribute('aTimeShift');
  const seeds = mesh.geometry.getAttribute('aSeed');
  return {
    sameGeometry: mesh.geometry === before.geometry,
    sameMaterial: mesh.material === before.material,
    sameUniforms: mesh.material.slime === before.uniforms,
    generation: window.__worldSim.generation().count === before.generation,
    shifts: Array.from(shifts.array), seeds: Array.from(seeds.array),
    vertices: mesh.geometry.getAttribute('position').count,
    version: shifts.version,
  };
});
const edit = async (field, value) => {
  await p.getByLabel(field, {exact:true}).fill(value);
  await p.keyboard.press('Tab');
  await p.waitForTimeout(250);
};
const initial = await read();
await edit('mesh.timeShiftSeconds', '17');
let actual = await read();
check('animation offset retains geometry material uniforms and world', actual.sameGeometry && actual.sameMaterial && actual.sameUniforms && actual.generation, actual);
check('every seeded animation offset reaches GPU attribute', actual.shifts.every((v,i) => Math.abs(v - actual.seeds[i]*17) < 0.000002) && actual.shifts.some(v=>v>0), actual);
const version = actual.version;
await p.waitForTimeout(500);
check('unchanged frames do not upload time shifts', (await read()).version === version);
await edit('mesh.widthSegments', '31');
actual = await read();
check('resolution rebuilds geometry while retaining material uniforms and world', !actual.sameGeometry && actual.vertices !== initial.vertices && actual.sameMaterial && actual.sameUniforms && actual.generation, actual);
check('new geometry preserves committed animation offsets', actual.shifts.every((v,i) => Math.abs(v - actual.seeds[i]*17) < 0.000002), actual);
await edit('mesh.timeShiftSeconds', '0');
actual = await read();
check('zero animation offset clears every instance', actual.shifts.every(v=>v===0), actual);
check('no browser errors',errors.length===0,errors)
}catch(e){checks.push({name:'completed',pass:false,detail:String(e)})}finally{writeFileSync(root+'actor-offset-browser-20260905.json',JSON.stringify({checks,errors},null,2));console.log(checks);await b.close()}if(checks.some(c=>!c.pass))process.exitCode=1;
