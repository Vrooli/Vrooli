import { readFileSync, existsSync, readdirSync } from "node:fs";
import { join } from "node:path";
import { execSync } from "node:child_process";
const LIB="scenarios/react-component-library/library";
const cmp=(a,b)=>a.localeCompare(b,undefined,{numeric:true});
const KINDS=readdirSync(LIB,{withFileTypes:true}).filter(e=>e.isDirectory()&&e.name!==".retired").map(e=>e.name);
const assets=new Map(), byLib=new Map();
for(const kind of KINDS) for(const name of readdirSync(join(LIB,kind),{withFileTypes:true}).filter(e=>e.isDirectory()).map(e=>e.name)){
  const mp=join(LIB,kind,name,"component.json"); if(!existsSync(mp))continue;
  const m=JSON.parse(readFileSync(mp,"utf8"));
  const vr=join(LIB,kind,name,"versions");
  const versions=existsSync(vr)?readdirSync(vr,{withFileTypes:true}).filter(e=>e.isDirectory()&&/^\d+\.\d+\.\d+$/.test(e.name)).map(e=>e.name).sort(cmp):[];
  const dep=new Set(m.deprecatedVersions||[]), ev=new Set(m.evictedVersions||[]);
  const rec={kind,name,libraryId:String(m.libraryId||`react-component-library:${name}`),latest:(m.latest||"").trim(),versions,
             active:versions.filter(v=>!dep.has(v)&&!ev.has(v))};
  assets.set(name,rec); byLib.set(rec.libraryId,rec);
}
const edgesExact=new Map(), edgesFloat=new Map(), all=new Set();
for(const[,a] of assets) for(const v of a.versions){
  const k=`${a.libraryId}@${v}`; all.add(k); edgesExact.set(k,[]); edgesFloat.set(k,[]);
  const lp=join(LIB,a.kind,a.name,"versions",v,"dependencies.json"); if(!existsSync(lp))continue;
  for(const d of (JSON.parse(readFileSync(lp,"utf8")).dependencies||[])){
    edgesExact.get(k).push(`${d.libraryId}@${d.version}`);
    // float: same major line, highest ACTIVE
    const t=byLib.get(d.libraryId);
    let fv=d.version;
    if(t){ const maj=d.version.split(".")[0];
      const best=t.active.filter(x=>x.startsWith(maj+".")).at(-1);
      if(best) fv=best; }
    edgesFloat.get(k).push(`${d.libraryId}@${fv}`);
  }
}
const roots=new Set();
for(const[,a] of assets) if(a.latest&&a.versions.includes(a.latest)) roots.add(`${a.libraryId}@${a.latest}`);
const ext=execSync(`grep -rhoE "@vrooli/react-component-library/[A-Za-z0-9_-]+/[0-9]+\\.[0-9]+\\.[0-9]+" scenarios/*/ui/src 2>/dev/null || true`,{encoding:"utf8"});
const extRoots=new Set();
for(const l of ext.split("\n")){ if(!l.trim())continue; const p=l.split("/"),a=assets.get(p[2]); if(!a)continue;
  const k=`${a.libraryId}@${p[3]}`; if(all.has(k)){roots.add(k); if(a.latest!==p[3])extRoots.add(k);} }
const close=(E)=>{const s=new Set(),st=[...roots];while(st.length){const k=st.pop();if(s.has(k))continue;s.add(k);for(const t of (E.get(k)||[]))if(!s.has(t))st.push(t);}return s;};
const A=close(edgesExact), B=close(edgesFloat);
const nonLatest=k=>{const[lib,v]=k.split("@");return byLib.get(lib)?.latest!==v;};
console.log(JSON.stringify({
  versions: all.size,
  externalNonLatestPins: extRoots.size,
  exact_reachable: A.size, exact_garbage: all.size-A.size,
  float_reachable: B.size, float_garbage: all.size-B.size,
  freedByFloatingIntraLibraryImports: A.size-B.size,
  reachable_nonLatest_exact: [...A].filter(nonLatest).length,
  reachable_nonLatest_float: [...B].filter(nonLatest).length,
  exact_garbage_keys: [...all].filter(k=>!A.has(k)).sort(cmp),
},null,2));
