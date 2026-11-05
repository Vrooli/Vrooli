// Normal workbench shadow edits exercise declared live and material update behavior.
import{chromium}from'playwright-core';import{writeFileSync}from'node:fs';import{installFixtureHook}from'./camera-fixtures.mjs';
const root=new URL('../../evidence/living-world/',import.meta.url).pathname;const b=await chromium.launch({executablePath:'/usr/bin/google-chrome',args:['--no-sandbox','--ignore-gpu-blocklist','--use-gl=angle','--use-angle=gl-egl']});const p=await b.newPage({viewport:{width:1400,height:900}});await installFixtureHook(p);const checks=[],errors=[];p.on('pageerror',e=>errors.push(e.message));p.on('console',m=>{if(m.type()==='error'&&/shader|WebGL|THREE/.test(m.text()))errors.push(m.text())});const check=(name,pass,detail)=>{checks.push({name,pass,detail});if(!pass)throw Error(name)};try{await p.goto('http://localhost:21235/world?scene=park&actors=25&seed=1&profile=high&diag=1&workbench=1');await p.waitForFunction(()=>window.__worldDiagnostics?.ready,null,{timeout:90000});await p.evaluate(()=>{const pending=[...window.__cameraFixtureRoots].map(r=>r.current);let render;while(pending.length){const f=pending.pop();for(const v of[f.memoizedProps?.store,f.memoizedProps?.value])if(typeof v?.getState==='function'&&v.getState().scene?.isScene)render=v;if(f.child)pending.push(f.child);if(f.sibling)pending.push(f.sibling)}window.__readSlime=()=>{let mesh;render.getState().scene.traverse(o=>{if(o.isInstancedMesh && o.material?.alphaMap?.isCanvasTexture)mesh=o});return mesh};const mesh=window.__readSlime();window.__slimeBefore={mesh,material:mesh.material,geometry:mesh.geometry,uniforms:mesh.material.slime,version:mesh.material.version,generation:window.__worldSim.generation().count}});await p.getByTitle('World Settings',{exact:true}).click();await p.getByText('World workbench',{exact:true}).click();await p.getByRole('button',{name:'actor',exact:true}).click();
await p.evaluate(() => { const b=window.__slimeBefore; b.texture=b.material.alphaMap; b.texture.addEventListener('dispose',()=>{window.__oldTextureDisposed=(window.__oldTextureDisposed??0)+1}) });
const read = () => p.evaluate(() => {
  const mesh = window.__readSlime(), before = window.__slimeBefore;
  return { sameGeometry:mesh.geometry===before.geometry, sameMesh:mesh===before.mesh,
    sameMaterial:mesh.material===before.material, sameTexture:mesh.material.alphaMap===before.texture,
    generation:window.__worldSim.generation().count===before.generation,
    opacity:mesh.material.opacity, textureWidth:mesh.material.alphaMap.image.width,
    disposed:window.__oldTextureDisposed??0 };
});
const edit = async (field,value,impact) => {
  const input=p.getByLabel(field,{exact:true});
  check(field+' declares '+impact, await input.locator('xpath=ancestor::tr').getByText(impact+' update',{exact:true}).count()===1);
  await input.fill(value);await p.keyboard.press('Tab');await p.waitForTimeout(250);
};
for(const [field,value] of [['shadow.spread','1.6'],['shadow.lift','0.09'],['shadow.hopShrink','0.3']]) {
  await edit(field,value,'live');const actual=await read();
  check(field+' retains mesh geometry material texture and world',actual.sameMesh&&actual.sameGeometry&&actual.sameMaterial&&actual.sameTexture&&actual.generation,actual);
}
await edit('shadow.opacity','0.31','material');let actual=await read();
check('opacity updates material without replacing texture',actual.opacity===.31&&actual.sameMaterial&&actual.sameTexture&&actual.sameMesh,actual);
await edit('shadow.textureSize','128','material');actual=await read();
check('texture resolution replaces only texture and disposes old texture once',actual.textureWidth===128&&!actual.sameTexture&&actual.sameMesh&&actual.sameMaterial&&actual.sameGeometry&&actual.generation&&actual.disposed===1,actual);
check('no browser errors',errors.length===0,errors)
}catch(e){checks.push({name:'completed',pass:false,detail:String(e)})}finally{writeFileSync(root+'actor-shadow-impact-browser-20260905.json',JSON.stringify({checks,errors},null,2));console.log(checks);await b.close()}if(checks.some(c=>!c.pass))process.exitCode=1;
