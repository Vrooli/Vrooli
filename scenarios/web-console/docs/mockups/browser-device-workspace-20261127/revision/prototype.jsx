import React, {useEffect, useRef, useState} from 'react';
import {createRoot} from 'react-dom/client';
import {Button} from '@vrooli/react-component-library/Button/2';
import {IconButton} from '@vrooli/react-component-library/IconButton';
import {
  ArrowDownLeft, ArrowLeft, ArrowRight, ArrowUpRight, Check, ChevronDown, ChevronRight,
  CircleHelp, Ellipsis, Globe2, Hand, House, LockKeyhole, MessageSquare,
  Moon, Plus, Power, RotateCw, ShoppingBag, Smartphone, Sun, TerminalSquare, X
} from 'lucide-react';
import './prototype.css';

const params = new URLSearchParams(location.search);
const startsInPicker = params.get('start') === 'picker';
const storeKey = 'aquila-browser-devices-design-v2';
const sampleTargets = [
  {id:'browser', name:'Studio browser', type:'Default browser', network:'online', driver:'you', tabs:['collection'], tab:'collection', cart:0},
  {id:'emulator', name:'Pixel 9', type:'Emulated device', network:'online', driver:null, tabs:['collection'], tab:'collection', cart:0},
  {id:'phone', name:'Desk phone', type:'Physical device', network:'offline', driver:null, tabs:['collection'], tab:'collection', cart:0}
];
const samplePages = {
  collection:['Studio collection','preview.kiln.test/collections/studio'],
  product:['The Studio Pour-over','preview.kiln.test/products/studio-pour-over'],
  cart:['Shopping bag','preview.kiln.test/cart'],
  shipping:['Shipping','preview.kiln.test/pages/shipping'],
  blank:['New tab','']
};
const agents = ['Codex','Claude'];

function readState() {
  try {
    const saved=JSON.parse(localStorage.getItem(storeKey));
    if(saved?.targets?.some(t=>t.id===saved.last)) return saved;
  } catch {}
  return {targets:sampleTargets.map(target=>startsInPicker&&target.id==='browser'?{...target,driver:null}:target), last:'browser'};
}

function Glyph({icon:Icon, size=18}) {return <Icon size={size} strokeWidth={1.7} aria-hidden="true"/>}
function Action({icon, children, variant='secondary', ...props}) {
  return <Button type="button" size="sm" shape="square" variant={variant} {...props}>{icon&&<Glyph icon={icon} size={16}/>}<span>{children}</span></Button>;
}
function IconAction({icon, label, ...props}) {
  return <IconButton type="button" size="sm" shape="square" surface="ghost" aria-label={label} title={label} {...props}><Glyph icon={icon}/></IconButton>;
}
function TargetIcon({target, className=''}) {
  return <span className={'targetIcon '+(target.type==='Emulated device'?'virtual':'')+' '+className}>
    <Glyph icon={target.type.includes('browser')?Globe2:Smartphone} size={19}/>
  </span>;
}
function Owner({target}) {
  const label=target.network==='offline'?'Offline':target.network==='saved'?'Saved':target.driver==='you'?'You have control':target.driver?target.driver+' is driving':'Ready';
  return <span className={'owner '+(target.network==='offline'?'offline':target.driver&&target.driver!=='you'?'agent':'')}><i/>{label}</span>;
}
function Modal({title,onClose,children}) {
  const ref=useRef(null);
  useEffect(()=>{ const node=ref.current; node.showModal(); return ()=>node.close(); },[]);
  return <dialog ref={ref} aria-label={title} onCancel={e=>{e.preventDefault();onClose()}}>
    <header><h2>{title}</h2><IconAction icon={X} label="Close" onClick={onClose}/></header>{children}
  </dialog>;
}
function TargetRows({targets,onOpen,currentId}) {
  return <div className="targetRows">{targets.map(t=><button type="button" className="targetRow" key={t.id} onClick={()=>onOpen(t.id)} aria-current={currentId===t.id?'true':undefined}>
    <TargetIcon target={t}/><span className="targetText"><strong>{t.name}</strong><small>{t.type}</small></span>
    <Owner target={t}/><span className="rowArrow"><Glyph icon={t.id===currentId?Check:ChevronRight} size={16}/></span>
  </button>)}</div>;
}

function Shop({page,cart,navigate,canInput,addToBag}) {
  return <div className="shop">
    <header className="shopHeader">
      <button type="button" className="shopWordmark" aria-label="Kiln and Co. home" disabled={!canInput} onClick={()=>navigate('collection')}>kiln &amp; co.</button>
      <nav aria-label="Shop">
        <button type="button" aria-current={page==='collection'?'page':undefined} onClick={()=>navigate('collection')} disabled={!canInput}>Collection</button>
        <button type="button" aria-current={page==='shipping'?'page':undefined} onClick={()=>navigate('shipping')} disabled={!canInput}>Shipping</button>
      </nav>
      <button type="button" className="bag" aria-label={`Shopping bag, ${cart} items`} onClick={()=>navigate('cart')} disabled={!canInput}><Glyph icon={ShoppingBag} size={18}/><span>{cart}</span></button>
    </header>
    {page==='collection'&&<>
      <section className="shopHero">
        <div className="shopCopy"><p className="shopEyebrow">The studio collection</p><h1>Daily rituals.<br/>Made by hand.</h1><p>Small-batch ceramics for the everyday.</p>
          <button type="button" className="shopAction" onClick={()=>navigate('product')} disabled={!canInput}>Explore the pour-over <Glyph icon={ArrowUpRight} size={15}/></button>
        </div>
        <div role="img" className="pottery" aria-label="Sunlit handmade ceramic pour-over, mug and clay bowl"/>
      </section>
      <div className="shopBottom"><span>Made in small batches</span><span>Glazed by hand</span><span>Designed to be used</span></div>
    </>}
    {page==='product'&&<section className="shopHero product">
      <div className="shopCopy">
        <button type="button" className="shopBack" onClick={()=>navigate('collection')} disabled={!canInput}><Glyph icon={ArrowLeft} size={14}/> Collection</button>
        <h1>The Studio<br/>Pour-over</h1><p>Hand-thrown stoneware in warm oat glaze.</p>
        <button className="shopAction" type="button" onClick={addToBag} disabled={!canInput}>Add to bag · $48 <Glyph icon={Plus} size={15}/></button>
      </div><div role="img" className="pottery" aria-label="Hand-thrown ceramic pour-over in warm oat glaze"/>
    </section>}
    {page==='cart'&&<section className="shopPlain"><h1>Your bag</h1>{cart?<div className="cartLine"><b>The Studio Pour-over</b><span>{cart} × $48</span></div>:<p>Your bag is empty.</p>}
      <button className="shopAction" type="button" onClick={()=>navigate('collection')} disabled={!canInput}>Back to collection <Glyph icon={ArrowRight} size={15}/></button>
    </section>}
    {page==='shipping'&&<section className="shopPlain"><h1>Shipping</h1><p>Orders ship in two to four business days. Every piece is wrapped and packed in the studio.</p></section>}
  </div>;
}

function App() {
  const [saved,setSaved]=useState(readState);
  const [screen,setScreen]=useState(startsInPicker?'picker':'viewer');
  const [modal,setModal]=useState(null);
  const [theme,setTheme]=useState(params.get('theme')||'dark');
  const [notice,setNotice]=useState('');
  const [addressDraft,setAddressDraft]=useState('');
  const [rotate,setRotate]=useState(false);
  const [history,setHistory]=useState([]);
  const [refresh,setRefresh]=useState(0);
  const target=saved.targets.find(t=>t.id===saved.last);
  const isDevice=!target.type.includes('browser');
  const page=target.tab;
  const owns=target.driver==='you';
  const agent=target.driver&&!owns?target.driver:null;
  const live=target.network==='online';
  const address=samplePages[page][1];

  useEffect(()=>{document.documentElement.dataset.theme=theme},[theme]);
  useEffect(()=>{try{localStorage.setItem(storeKey,JSON.stringify(saved))}catch{}},[saved]);
  useEffect(()=>{setAddressDraft(address)},[address]);

  function patch(id,update) {
    setSaved(value=>({...value,targets:value.targets.map(t=>t.id===id?{...t,...update}:t)}));
  }
  function open(id) {
    setSaved(value=>({...value,last:id,targets:value.targets.map(t=>t.id===id&&t.network==='online'&&!t.driver?{...t,driver:'you'}:t)}));
    setScreen('viewer');
    setModal(null);
    setHistory([]);
    setRotate(false);
  }
  function enter() {
    if(startsInPicker) setScreen('picker');
    else open(saved.last);
  }
  function navigate(next) {
    if(!owns) return;
    setHistory(h=>[...h,page]);
    patch(target.id,{tab:next,tabs:target.tabs.includes(next)?target.tabs:[...target.tabs,next]});
  }
  function back() {
    if(!owns||!history.length) return;
    patch(target.id,{tab:history[history.length-1]});
    setHistory(h=>h.slice(0,-1));
  }
  function go(event) {
    event.preventDefault();
    if(!owns) return;
    const value=addressDraft.trim().replace(/^https?:\/\//,'');
    const next=Object.keys(samplePages).find(key=>samplePages[key][1]===value);
    if(next) navigate(next);
    else {setNotice('Use a preview.kiln.test address in this prototype.');setAddressDraft(address)}
  }
  function share(name) {
    patch(target.id,{driver:name,paused:null});
    setModal(null);
    setNotice(name+' now controls '+target.name+'.');
  }
  function take() {
    patch(target.id,{driver:'you',paused:agent});
    setNotice(agent?'You took control from '+agent+'.':'You have control.');
  }
  function finish() {
    const previous=agent||target.paused;
    patch(target.id,{driver:null,paused:null});
    setModal(null);
    setNotice(previous+' ended the sample run. The page and bag are still here.');
  }
  function sleep() {
    patch(target.id,{network:'saved',driver:null,paused:null});
    setModal(null);
    setNotice(target.name+' was saved.');
  }
  function add(type) {
    const id='new-'+(saved.targets.length+1);
    const name=type==='Browser'?'New browser':'Android emulator';
    setSaved(value=>({last:id,targets:[...value.targets,{id,name,type:type==='Browser'?'Managed browser':'Emulated device',network:'online',driver:'you',tab:type==='Browser'?'blank':'collection',tabs:[type==='Browser'?'blank':'collection'],cart:0}]}));
    setModal(null);setScreen('viewer');setNotice(name+' opened.');setHistory([]);setRotate(false);
  }

  const driverActions=<>
    {live&&agent&&<Action variant="primary" icon={Hand} onClick={take}>Take control</Action>}
    {live&&!target.driver&&<Action variant="primary" icon={Hand} onClick={take}>{isDevice?'Control device':'Use browser'}</Action>}
    {live&&owns&&target.paused&&<Action variant="primary" icon={ArrowDownLeft} onClick={()=>share(target.paused)}>Return to {target.paused}</Action>}
    {live&&owns&&!target.paused&&<Action variant="primary" icon={ArrowUpRight} onClick={()=>setModal('share')}>Share</Action>}
    {target.network==='saved'&&<Action variant="primary" icon={Power} onClick={()=>{patch(target.id,{network:'online',driver:'you'});setNotice('Saved browser reopened.')}}>Resume browser</Action>}
    {target.network==='offline'&&<Action icon={RotateCw} onClick={()=>setNotice('The desk phone is still offline.')}>Check again</Action>}
  </>;

  return <div className={'app '+(screen==='viewer'?'appViewing':'')}>
    <aside className="rail" aria-label="Aquila navigation">
      <div className="railTop">
        <div className="aquilaLogo"><span className="logoMark" aria-hidden="true"/></div>
        <IconAction icon={TerminalSquare} label="Workspace" onClick={()=>setScreen('workspace')} aria-current={screen==='workspace'?'page':undefined}/>
        <IconAction icon={Globe2} label="Browser & devices" onClick={enter} aria-current={screen!=='workspace'?'page':undefined}/>
      </div>
      <IconAction icon={theme==='dark'?Sun:Moon} label={theme==='dark'?'Use light theme':'Use dark theme'} onClick={()=>setTheme(theme==='dark'?'light':'dark')}/>
    </aside>
    <header className="globalBar">
      <div className="globalLeft"><span className="mobileLogo logoMark" aria-hidden="true"/><button type="button" className={screen==='workspace'?'globalSection active':'globalSection'} onClick={()=>setScreen('workspace')}>Workspace</button><Glyph icon={ChevronRight} size={14}/><button type="button" className={screen!=='workspace'?'globalSection active':'globalSection'} onClick={enter}>Browser &amp; devices</button></div>
      <div className="globalRight"><span className="demoLabel">Design preview</span><div className="me" aria-label="Your account">M</div><span className="mobileTheme"><IconAction icon={theme==='dark'?Sun:Moon} label={theme==='dark'?'Use light theme':'Use dark theme'} onClick={()=>setTheme(theme==='dark'?'light':'dark')}/></span></div>
    </header>
    {screen==='picker'&&<main className="inventory">
      <header className="inventoryHeader"><div><h1>Browser &amp; devices</h1><p>Open a browser or device.</p></div><Action icon={Plus} variant="primary" onClick={()=>setModal('add')}>Add target</Action></header>
      <div className="inventoryBody">
        <section className="inventoryList" aria-label="Available browsers and devices">
          <div className="inventoryListHeader"><span>Target</span><span>Status</span></div>
          <TargetRows targets={saved.targets} onOpen={open}/>
        </section>
        <aside className="resume">
          <div className="resumeArt">{target.network!=='offline'&&['collection','product'].includes(page)?<div className="resumeImage"/>:<TargetIcon target={target}/>} </div>
          <div className="resumeCaption"><small>Last opened</small><h2>{target.name}</h2><p>{samplePages[target.tab][0]}</p><Action icon={ArrowRight} onClick={()=>open(target.id)}>Open target</Action></div>
        </aside>
      </div>
    </main>}
    {screen==='workspace'&&<main className="workspace">
      <header><span className="workspaceIcon"><Glyph icon={MessageSquare}/></span><div><h1>Storefront QA</h1><p>Example conversation</p></div></header>
      <div className="thread">
        <div className="sampleMessage"><strong>You</strong><p>Let’s check the studio collection and the product page on mobile.</p></div>
        <button type="button" className="linkedTarget" onClick={()=>open('browser')}><TargetIcon target={sampleTargets[0]}/><span><b>Studio browser</b><small>Open shared browser</small></span><Glyph icon={ArrowUpRight}/></button>
      </div>
    </main>}
    {screen==='viewer'&&<main className="viewer">
      <header className="viewerHeader">
        <button type="button" className="targetButton" aria-haspopup="dialog" aria-label={'Switch target, '+target.name} onClick={()=>setModal('targets')}>
          <TargetIcon target={target}/><span><strong>{target.name}</strong><small>{target.type}</small></span><Glyph icon={ChevronDown} size={16}/>
        </button>
        <div className="viewerHeaderActions"><Owner target={target}/><span className="desktopActions">{driverActions}</span><IconAction icon={Ellipsis} label="Target options" onClick={()=>setModal('options')}/></div>
      </header>
      <div className={'stage '+(isDevice?'deviceStage':'browserStage')+(rotate?' rotated':'')}>
        {!live?<section className="empty">
          <TargetIcon target={target}/><h1>{target.network==='offline'?target.name+' is offline':'Browser saved'}</h1>
          <p>{target.network==='offline'?'Bring the phone online to reconnect.':'Resume when you’re ready to pick up here.'}</p>
        </section>:!isDevice?<div className="browser">
          <div className="browserTabs">
            {target.tabs.map(tab=><button type="button" key={tab} className={tab===page?'selected':''} aria-pressed={tab===page} disabled={!owns} onClick={()=>{setHistory(h=>[...h,page]);patch(target.id,{tab})}}><Glyph icon={Globe2} size={14}/><span>{samplePages[tab][0]}</span></button>)}
            <IconAction icon={Plus} label="New tab" disabled={!owns} onClick={()=>navigate('blank')}/>
          </div>
          <form className="browserAddress" onSubmit={go}>
            <IconAction icon={ArrowLeft} label="Go back" disabled={!owns||!history.length} onClick={back}/>
            <IconAction icon={RotateCw} label="Reload page" disabled={!owns} onClick={()=>setRefresh(n=>n+1)}/>
            <div className="url"><Glyph icon={LockKeyhole} size={14}/><input type="text" aria-label="Browser address" autoCapitalize="none" spellCheck="false" placeholder="Enter a preview.kiln.test address" value={addressDraft} readOnly={!owns} onChange={e=>setAddressDraft(e.target.value)}/></div>
          </form>
          <div className="webContent">{page==='blank'?<section className="empty newPage"><Glyph icon={Globe2} size={32}/><h1>New tab</h1><p>Enter an address to open a page.</p><Action onClick={()=>navigate('collection')} disabled={!owns}>Open sample storefront</Action></section>:<Shop key={refresh} page={page} cart={target.cart} navigate={navigate} canInput={owns} addToBag={()=>{patch(target.id,{cart:target.cart+1});setNotice('Added to bag.')}}/>}</div>
        </div>:<div className="device">
          <div className="deviceStatus"><span>9:41</span><span>●●●</span></div>
          <div className="webContent"><Shop page={page} cart={target.cart} navigate={navigate} canInput={owns} addToBag={()=>{patch(target.id,{cart:target.cart+1});setNotice('Added to bag.')}}/></div>
          <div className="deviceKeys"><IconAction icon={ArrowLeft} label="Device back" onClick={back} disabled={!owns||!history.length}/><IconAction icon={House} label="Device home" onClick={()=>navigate('collection')} disabled={!owns}/><IconAction icon={RotateCw} label="Rotate device" onClick={()=>setRotate(v=>!v)} disabled={!owns}/></div>
        </div>}
      </div>
      <footer className="viewerFooter">
        <div className="footerStatus"><span className={'footerDot '+(!live?'quiet':'')}/><span>{live?(agent?'Watching live':owns?'Live · Input goes to '+target.name:'Available to use'):(target.network==='offline'?'Waiting for device':'Saved tabs and profile')}</span></div>
        <button type="button" className="conversationLink" onClick={()=>setModal('conversation')}><Glyph icon={MessageSquare} size={15}/> <span>Conversation</span></button>
        <div className="mobileActions">{driverActions}</div>
      </footer>
    </main>}
    {modal==='targets'&&<Modal title="Switch target" onClose={()=>setModal(null)}><TargetRows targets={saved.targets} currentId={target.id} onOpen={open}/><div className="modalFoot"><Action onClick={()=>{setScreen('picker');setModal(null)}}>See all targets</Action><Action variant="primary" icon={Plus} onClick={()=>setModal('add')}>Add target</Action></div></Modal>}
    {modal==='add'&&<Modal title="Add target" onClose={()=>setModal(null)}><div className="modalRows">
      <button type="button" onClick={()=>add('Browser')}><Glyph icon={Globe2}/><span><b>New browser</b><small>Start with separate tabs and profile.</small></span><Glyph icon={ChevronRight}/></button>
      <button type="button" onClick={()=>add('Emulator')}><Glyph icon={Smartphone}/><span><b>Start an emulator</b><small>Open a virtual Android device.</small></span><Glyph icon={ChevronRight}/></button>
      <button type="button" onClick={()=>setModal('pair')}><Glyph icon={Smartphone}/><span><b>Pair a physical device</b><small>Connect supported hardware.</small></span><Glyph icon={ChevronRight}/></button>
    </div></Modal>}
    {modal==='pair'&&<Modal title="Pair a physical device" onClose={()=>setModal(null)}><p className="modalCopy">Pairing opens Device Control in Aquila. This preview includes one example physical device.</p><div className="modalFoot"><Action onClick={()=>setModal('add')}>Back</Action><Action variant="primary" onClick={()=>open('phone')}>View Desk phone</Action></div></Modal>}
    {modal==='share'&&<Modal title={'Share '+target.name} onClose={()=>setModal(null)}>
      <p className="modalCopy">Choose who will control this {isDevice?'device':'browser'}. You can watch and take over at any time.</p>
      <div className="modalRows">{agents.map(name=><button type="button" key={name} onClick={()=>share(name)}><span className={'agentGlyph '+name.toLowerCase()}>{name==='Codex'?'C':'✳'}</span><span><b>{name}</b><small>Give control</small></span><Glyph icon={ArrowRight}/></button>)}</div>
    </Modal>}
    {modal==='options'&&<Modal title={target.name} onClose={()=>setModal(null)}><div className="modalRows">
      {live&&owns&&target.paused&&<button type="button" onClick={()=>setModal('share')}><Glyph icon={ArrowUpRight}/><span><b>Share with another agent</b></span><Glyph icon={ChevronRight}/></button>}
      {(agent||target.paused)&&<button type="button" onClick={finish}><Glyph icon={Power}/><span><b>{'End '+(agent||target.paused)+'’s example run'}</b><small>Keep the target open.</small></span><Glyph icon={ChevronRight}/></button>}
      {live&&owns&&!target.paused&&!isDevice&&<button type="button" onClick={sleep}><Glyph icon={Moon}/><span><b>Save and sleep browser</b><small>Keep the tabs and profile for later.</small></span><Glyph icon={ChevronRight}/></button>}
      <button type="button" onClick={()=>{setModal(null);setScreen('picker')}}><Glyph icon={Globe2}/><span><b>See all targets</b></span><Glyph icon={ChevronRight}/></button>
    </div></Modal>}
    {modal==='conversation'&&<Modal title="Storefront QA" onClose={()=>setModal(null)}><div className="conversationBody"><p><strong>You</strong><span>Let’s check the studio collection and the product page on mobile.</span></p><p><strong>{agent||target.paused||'Codex'}</strong><span>I can use {target.name} when you share it.</span></p></div><div className="modalFoot"><Action onClick={()=>{setModal(null);setScreen('workspace')}}>Open conversation</Action>{live&&owns&&<Action variant="primary" icon={ArrowUpRight} onClick={()=>setModal('share')}>Share target</Action>}</div></Modal>}
    {notice&&<div className="toast" role="status"><Glyph icon={CircleHelp} size={17}/><span>{notice}</span><IconAction icon={X} label="Dismiss message" onClick={()=>setNotice('')}/></div>}
  </div>;
}
createRoot(document.getElementById('root')).render(<App/>);
