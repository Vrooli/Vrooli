#!/usr/bin/env python3
"""Measure whether indexed component stories paint visible specimen content."""
import argparse, json, pathlib, sys
from urllib.request import Request, urlopen
from playwright.sync_api import sync_playwright

BASE = 'http://localhost:23906'; SCENARIO = pathlib.Path(__file__).resolve().parents[2]
LIST = SCENARIO / 'data' / 'components-list.json'
MEASURE = r"""
() => {
 const root=document.querySelector('[data-testid="component-harness-root"]'); if(!root)return {err:'no harness root'};
 const status=root.getAttribute('data-rcl-story-status'), state=root.getAttribute('data-experience-state'); let passed=null;
 try{passed=JSON.parse(document.getElementById('rcl-story-result')?.textContent||'{}').passed}catch(e){}
 const stage=document.querySelector('[data-preview-stage]')||document.querySelector('[data-preview-capture-boundary]'); if(!stage)return {status,state,passed,err:'no stage'};
 const chrome=el=>[...el.classList].some(c=>c.startsWith('rcl-preview-')); let painted=0,l=1e9,t=1e9,r=-1e9,b=-1e9,textLen=0;
 for(const el of stage.querySelectorAll('*')){if(el.hasAttribute('data-preview-readiness-marker')||chrome(el)||el.closest('[data-preview-readiness-marker]'))continue;const cs=getComputedStyle(el);if(cs.display==='none'||cs.visibility==='hidden'||parseFloat(cs.opacity)===0)continue;const box=el.getBoundingClientRect();if(box.width<1||box.height<1)continue;const bg=cs.backgroundColor&&cs.backgroundColor!=='rgba(0, 0, 0, 0)';const bd=['borderTopWidth','borderRightWidth','borderBottomWidth','borderLeftWidth'].some(k=>parseFloat(cs[k])>0);const ownText=[...el.childNodes].some(n=>n.nodeType===3&&n.textContent.trim().length);const media=['SVG','IMG','CANVAS','VIDEO','svg','img','canvas'].includes(el.tagName);if(!(bg||bd||ownText||media))continue;painted++;if(ownText)textLen+=el.textContent.trim().length;l=Math.min(l,box.left);t=Math.min(t,box.top);r=Math.max(r,box.right);b=Math.max(b,box.bottom)}
 const w=painted?Math.round(r-l):0,h=painted?Math.round(b-t):0; return {status,state,passed,painted,w,h,area:w*h,textLen};
}
"""
def component_list():
    if LIST.exists(): return json.loads(LIST.read_text()).get('components', [])
    request=Request(BASE+'/vrooli.react_component_library.v1.components.ComponentsService/ListComponents',data=json.dumps({'limit':2000}).encode(),headers={'Content-Type':'application/json'})
    with urlopen(request,timeout=30) as response: return json.load(response).get('components', [])
parser=argparse.ArgumentParser(); parser.add_argument('--output',default='/tmp/rcl-render-audit.json'); args=parser.parse_args(); results=[]
with sync_playwright() as playwright:
    browser=playwright.chromium.launch(executable_path='/usr/bin/google-chrome',args=['--no-sandbox','--disable-dev-shm-usage']); page=browser.new_context(viewport={'width':1200,'height':800}).new_page(); components=component_list()
    for index,component in enumerate(components):
        record={'name':component.get('displayName'),'id':component.get('id'),'rung':component.get('catalogRungName'),'kind':component.get('assetKind'),'maturity':component.get('catalogRung')}
        try:
            page.goto(f"{BASE}/assets/{component['id']}",wait_until='domcontentloaded',timeout=20000); page.wait_for_timeout(2300); frames=[frame for frame in page.frames if 'harness.html' in frame.url]; record['result']=frames[0].evaluate(MEASURE) if frames else {'err':'no harness iframe'}
        except Exception as error: record['result']={'err':str(error)[:200]}
        results.append(record)
        if (index+1)%25==0: print(f'  ...{index+1}/{len(components)}',file=sys.stderr,flush=True)
    browser.close()
path=pathlib.Path(args.output); path.parent.mkdir(parents=True,exist_ok=True); path.write_text(json.dumps(results,indent=1)); print('wrote',path,len(results))
