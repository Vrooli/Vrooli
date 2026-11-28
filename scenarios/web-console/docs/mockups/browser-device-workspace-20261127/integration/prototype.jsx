import React, {useEffect, useRef, useState} from 'react';
import {createRoot} from 'react-dom/client';
import {Button} from '@vrooli/react-component-library/Button/2';
import {IconButton} from '@vrooli/react-component-library/IconButton';
import {
  ArrowDownLeft, ArrowLeft, ArrowRight, ArrowUpRight, Check, ChevronDown, ChevronRight,
  CircleHelp, Ellipsis, Globe2, Hand, House, LockKeyhole, MessageSquare,
  Archive, ArrowDownUp, CheckSquare, ListFilter, Menu, Mic, Moon, Paperclip, Plus, Power, RotateCw, Search, Settings, ShieldCheck, ShoppingBag, Smartphone, Sun, TerminalSquare, ArrowUp, X
} from 'lucide-react';
import '../revision/prototype.css';
import './integration.css';
import {ActivityTimeline, HistoryFrame, initialActivity} from './ActivityTimeline.jsx';

const params = new URLSearchParams(location.search);
const storeKey = 'aquila-browser-devices-integrated';
const sampleTargets = [
  {id:'browser', name:'Studio browser', type:'Default browser', network:'online', driver:null, tabs:['collection','product'], tab:'product', cart:0, activity:initialActivity},
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
  return {targets:sampleTargets,last:'browser'};
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
function Modal({title,onClose,children,drawer=false}) {
  const ref=useRef(null);
  useEffect(()=>{ const node=ref.current; node.showModal(); return ()=>node.close(); },[]);
  return <dialog ref={ref} className={drawer?'drawerModal':undefined} aria-label={title} onCancel={e=>{e.preventDefault();onClose()}}>
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
  const [screen,setScreen]=useState(params.get('start')==='browser'?'viewer':'workspace');
  const [mobileSidebar,setMobileSidebar]=useState(false);
  const [snapshot,setSnapshot]=useState(null);
  const [conversation,setConversation]=useState('Storefront review');
  const [draft,setDraft]=useState('');
  const [messages,setMessages]=useState([]);
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
  const isDriver=target.driver==='you';
  const owns=isDriver&&!snapshot;
  const agent=target.driver&&!isDriver?target.driver:null;
  const live=target.network==='online';
  const address=samplePages[page][1];

  useEffect(()=>{document.documentElement.dataset.theme=theme},[theme]);
  useEffect(()=>{try{localStorage.setItem(storeKey,JSON.stringify(saved))}catch{}},[saved]);
  useEffect(()=>{setAddressDraft(address)},[address]);

  useEffect(()=>{
    const timer=window.setInterval(()=>{
      setSaved(value=>{
        const active=value.targets.filter(t=>t.driver&&t.driver!=='you'&&t.network==='online'&&t.job<3);
        if(!active.length) return value;
        return {...value,targets:value.targets.map(t=>{
          if(!active.includes(t)) return t;
          const [page,action]=[['collection','Opened storefront'],['product','Opened product'],['shipping','Checked shipping']][t.job];
          const item={id:t.id+'-'+Date.now()+'-'+t.job,page,action,agent:t.driver,time:new Intl.DateTimeFormat('en-US',{hour:'numeric',minute:'numeric'}).format(new Date())};
          return {...t,job:t.job+1,tab:page,tabs:t.tabs.includes(page)?t.tabs:[...t.tabs,page],activity:[...(t.activity||[]),item]};
        })};
      });
    },4600);
    return ()=>window.clearInterval(timer);
  },[]);

  function patch(id,update) {
    setSaved(value=>({...value,targets:value.targets.map(t=>t.id===id?{...t,...update}:t)}));
  }
  function open(id) {
    setSaved(value=>({...value,last:id,targets:value.targets.map(t=>t.id===id&&t.network==='online'&&!t.driver?{...t,driver:'you'}:t)}));
    setScreen('viewer');
    setModal(null);
    setMobileSidebar(false);
    setSnapshot(null);
    setHistory([]);
    setRotate(false);
  }
  function enter() {open(saved.last);}
  function openConversation(name=conversation) {setConversation(name);setScreen('workspace');setMobileSidebar(false);}
  function sendMessage(event) {
    event.preventDefault();
    if(!draft.trim()) return;
    setMessages(items=>[...items,{conversation,text:draft.trim()}]);setDraft('');
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
    patch(target.id,{driver:name,paused:null,job:target.paused===name?target.job:0});
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
    const typeName={Browser:'Managed browser',Emulator:'Emulated device',Physical:'Physical device'}[type];
    const name={Browser:'New browser',Emulator:'Android emulator',Physical:'Paired phone'}[type];
    const tab=type==='Browser'?'blank':'collection';
    setSaved(value=>({last:id,targets:[...value.targets,{id,name,type:typeName,network:'online',driver:'you',tab,tabs:[tab],cart:0}]}));
    setModal(null);setScreen('viewer');setSnapshot(null);setNotice(name+' opened.');setHistory([]);setRotate(false);
  }

  const driverActions=snapshot?<Action variant="primary" icon={ArrowRight} onClick={()=>setSnapshot(null)}>{live?'Return to Live':'Return to current view'}</Action>:<>
    {live&&agent&&<Action variant="primary" icon={Hand} onClick={take}>Take control</Action>}
    {live&&!target.driver&&<Action variant="primary" icon={Hand} onClick={take}>{isDevice?'Control device':'Use browser'}</Action>}
    {live&&owns&&target.paused&&<Action variant="primary" icon={ArrowDownLeft} onClick={()=>share(target.paused)}>Return to {target.paused}</Action>}
    {live&&owns&&!target.paused&&<Action variant="primary" icon={ArrowUpRight} onClick={()=>setModal('share')}>Share</Action>}
    {target.network==='saved'&&<Action variant="primary" icon={Power} onClick={()=>{patch(target.id,{network:'online',driver:'you'});setNotice('Saved browser reopened.')}}>Resume browser</Action>}
    {target.network==='offline'&&<Action icon={RotateCw} onClick={()=>setNotice('The desk phone is still offline.')}>Check again</Action>}
  </>;

  const sidebarContent=<>
    <header className="sessionsHeader"><span className="logoMark"/><div><strong>Sessions</strong><small>2 sessions</small></div><IconAction icon={Plus} label="New session" onClick={()=>setModal('launch')}/><IconAction icon={mobileSidebar?X:Settings} label={mobileSidebar?'Close Sessions':'Settings'} onClick={()=>mobileSidebar?setMobileSidebar(false):setModal('settings')}/></header>
    <div className="sortLine"><ArrowDownUp size={13} aria-hidden="true"/><span>Sort</span><span className="sortValue">Manual <ChevronDown size={13} aria-hidden="true"/></span></div>
    <div className="sessionRows">{['Storefront review','Release checklist'].map(name=><button type="button" key={name} className="sessionRow" aria-current={name===conversation&&screen!=='viewer'?'page':undefined} onClick={()=>openConversation(name)}><i/><span><strong>{name}</strong><small><MessageSquare size={11}/> {name==='Storefront review'?'Visited just now · Messages':'Messages'}</small></span></button>)}</div>
    <div className="sessionsFooter">
      <button type="button" className="globalBrowser" onClick={enter} aria-current={screen==='viewer'?'page':undefined}><Glyph icon={Globe2} size={17}/> Browser &amp; devices <Glyph icon={ChevronRight} size={15}/></button>
      <button type="button" className="archiveRow" onClick={()=>setModal('archive')}><Glyph icon={Archive} size={16}/> Archive <small>0</small></button>
    </div>
  </>;
  return <div className="aquilaShell">
    <aside className="sessionsDesktop" aria-label="Sessions">{sidebarContent}</aside>
    <header className="aquilaPhoneBar">
      <IconAction icon={Menu} label="Open Sessions" onClick={()=>setMobileSidebar(true)}/>
      <span>{screen==='viewer'?'Browser & devices':conversation}</span>
      <IconAction icon={Plus} label="New session" onClick={()=>setModal('launch')}/>
      <IconAction icon={Settings} label="Settings" onClick={()=>setModal('settings')}/>
      <IconAction icon={ShieldCheck} label="Approvals" onClick={()=>setModal('approvals')}/>
    </header>
    {mobileSidebar&&<Modal title="Sessions" drawer onClose={()=>setMobileSidebar(false)}><aside className="sessionsMobile" aria-label="Sessions">{sidebarContent}</aside></Modal>}
    {screen==='picker'&&<main className="inventory">
      <header className="inventoryHeader"><div><h1>Browser &amp; devices</h1><p>Open a browser or device.</p></div><Action icon={Plus} variant="primary" onClick={()=>setModal('add')}>Add or connect</Action></header>
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
    {(screen==='workspace'||screen==='terminal')&&<>
      <main className="workspace shellPane">
        <div className="viewTools">
          <button type="button" className="messageSearch" onClick={()=>setModal('search')}><Glyph icon={Search} size={16}/><span>Search 3 messages</span></button>
          <IconAction icon={ListFilter} label="Filter messages" onClick={()=>setModal('filter')}/>
          <IconAction icon={CheckSquare} label="Select messages" onClick={()=>setModal('select')}/>
          <IconAction icon={Globe2} label="Open browser view" onClick={()=>conversation==='Storefront review'?open('browser'):setScreen('picker')}/>
          <IconAction icon={screen==='terminal'?MessageSquare:TerminalSquare} label={screen==='terminal'?'Switch to Messages':'Switch to Terminal'} onClick={()=>setScreen(screen==='terminal'?'workspace':'terminal')}/>
        </div>
        <div className="conversationScroll">
          {screen==='workspace'?<div className="messageList">
            <article className="chatMessage you"><div><b>You</b><time>9:00 AM</time></div><p>Compare the Studio Pottery collection across desktop and mobile.</p></article>
            <article className="chatMessage"><div><b>Agent</b><time>9:01 AM</time></div><p>I’ll check the product page, compare the mobile layout, and keep the browser ready for you to review.</p></article>
            <article className="chatMessage you"><div><b>You</b><time>9:02 AM</time></div><p>Start with the pour-over set. Leave the product page open so I can pick up from there.</p></article>
            {messages.filter(message=>message.conversation===conversation).map((message,i)=><article className="chatMessage you" key={i}><div><b>You</b></div><p>{message.text}</p></article>)}
          </div>:<pre className="sampleTerminal"><span>$</span> …</pre>}
        </div>
      </main>
      <div className="messagesFooter">
        <form className="messageComposer" onSubmit={sendMessage}>
          <input aria-label="Message" placeholder="Type command…" value={draft} onChange={event=>setDraft(event.target.value)}/>
          <Button type="submit" aria-label="Send" variant="primary" size="sm" disabled={!draft.trim()}><Glyph icon={ArrowUp}/></Button>
        </form>
        <div className="messageKeys">
          <IconAction icon={Ellipsis} label="More composer controls" onClick={()=>setModal('composer')}/>
          <IconAction icon={ListFilter} label="Snippets" onClick={()=>setModal('snippets')}/>
          <IconAction icon={Mic} label="Voice input" onClick={()=>setModal('voice')}/>
          <IconAction icon={Paperclip} label="Attach file" onClick={()=>setModal('attach')}/>
        </div>
      </div>
    </>}
    {screen==='viewer'&&<main className="viewer shellPane">
      <header className="viewerHeader">
        <button type="button" className="targetButton" aria-haspopup="dialog" aria-label={'Switch target, '+target.name} onClick={()=>setModal('targets')}>
          <TargetIcon target={target}/><span><strong>{target.name}</strong><small>{target.type}</small></span><Glyph icon={ChevronDown} size={16}/>
        </button>
        <div className="viewerHeaderActions"><Owner target={target}/><span className="desktopActions">{driverActions}</span><IconAction icon={Ellipsis} label="Target options" disabled={Boolean(snapshot)} onClick={()=>setModal('options')}/></div>
      </header>
      <div className={'stage '+(snapshot?'reviewStage ':'')+(isDevice?'deviceStage':'browserStage')+(rotate?' rotated':'')}>
        {snapshot?<div className="snapshot">
          <div className="snapshotHeader"><div><strong>Reviewing screenshot</strong><span>{snapshot.agent} · {snapshot.time} · {snapshot.action}</span></div><span className="readOnlyTag">Read-only</span></div>
          <HistoryFrame item={snapshot} device={isDevice}/>
        </div>:!live?<section className="empty">
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
      <ActivityTimeline entries={target.activity||[]} selected={snapshot} onSelect={setSnapshot} agent={agent} live={live}/>
      <footer className="viewerFooter">
        <div className="paneViews">
          <IconAction icon={MessageSquare} label="Back to Messages" onClick={()=>openConversation()}/>
          <IconAction icon={TerminalSquare} label="Switch to Terminal" onClick={()=>setScreen('terminal')}/>
        </div>
        <div className="footerStatus"><span className={'footerDot '+(!live?'quiet':'')}/><span>{snapshot?'Reviewing history':live?(agent?'Watching live':owns?'Live · Input goes to '+target.name:'Available to use'):(target.network==='offline'?'Waiting for device':'Saved tabs and profile')}</span></div>
        <div className="mobileActions">{driverActions}</div>
      </footer>
    </main>}
    {modal==='targets'&&<Modal title="Switch target" onClose={()=>setModal(null)}><TargetRows targets={saved.targets} currentId={target.id} onOpen={open}/><div className="modalFoot"><Action onClick={()=>{setScreen('picker');setModal(null)}}>See all targets</Action><Action variant="primary" icon={Plus} onClick={()=>setModal('add')}>Add or connect</Action></div></Modal>}
    {modal==='add'&&<Modal title="Add or connect" onClose={()=>setModal(null)}><div className="modalRows">
      <button type="button" onClick={()=>add('Browser')}><Glyph icon={Globe2}/><span><b>New browser</b><small>Start with separate tabs and profile.</small></span><Glyph icon={ChevronRight}/></button>
      <button type="button" onClick={()=>setModal('emulator')}><Glyph icon={Smartphone}/><span><b>Start an emulator</b><small>Choose a virtual Android device.</small></span><Glyph icon={ChevronRight}/></button>
      <button type="button" onClick={()=>setModal('connect')}><Glyph icon={Smartphone}/><span><b>Connect a device</b><small>Reconnect or pair a physical device.</small></span><Glyph icon={ChevronRight}/></button>
    </div></Modal>}
    {modal==='emulator'&&<Modal title="Start an emulator" onClose={()=>setModal(null)}><div className="modalRows">
      <button type="button" onClick={()=>add('Emulator')}><Glyph icon={Smartphone}/><span><b>Pixel 9 · Android 16</b><small>Start a new virtual device</small></span><Glyph icon={ArrowRight}/></button>
    </div><div className="modalFoot"><Action onClick={()=>setModal('add')}>Back</Action></div></Modal>}
    {modal==='connect'&&<Modal title="Connect a device" onClose={()=>setModal(null)}><div className="modalRows">
      <button type="button" onClick={()=>open('phone')}><Glyph icon={Smartphone}/><span><b>Desk phone</b><small>Previously paired · Offline</small></span><Glyph icon={ChevronRight}/></button>
      <button type="button" onClick={()=>setModal('pair')}><Glyph icon={Plus}/><span><b>Pair a new device</b><small>Open the Device Control pairing flow</small></span><Glyph icon={ArrowRight}/></button>
    </div><div className="modalFoot"><Action onClick={()=>setModal('add')}>Back</Action></div></Modal>}
    {modal==='pair'&&<Modal title="Pair a new device" onClose={()=>setModal(null)}><p className="modalCopy">Device Control verifies the device before it can be opened in Aquila. This prototype shows an example pairing result.</p><div className="modalFoot"><Action onClick={()=>setModal('connect')}>Back</Action><Action variant="primary" onClick={()=>add('Physical')}>Pair example phone</Action></div></Modal>}
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
    {['launch','settings','approvals','archive','search','filter','select','composer','snippets','attach','voice'].includes(modal)&&<Modal title={{launch:'New session',settings:'Settings',approvals:'Approvals',archive:'Archive',search:'Search messages',filter:'Filter messages',select:'Select messages',composer:'Composer controls',snippets:'Snippets',attach:'Attach file',voice:'Voice input'}[modal]} onClose={()=>setModal(null)}>
      {modal==='settings'?<div className="modalRows"><button type="button" onClick={()=>setTheme(theme==='dark'?'light':'dark')}><Glyph icon={theme==='dark'?Sun:Moon}/><span><b>{theme==='dark'?'Use light theme':'Use dark theme'}</b></span><Glyph icon={ChevronRight}/></button></div>:<p className="modalCopy shellFixtureCopy">This is an existing Aquila control. The preview focuses on Browser &amp; devices.</p>}
    </Modal>}
    {notice&&<div className="toast" role="status"><Glyph icon={CircleHelp} size={17}/><span>{notice}</span><IconAction icon={X} label="Dismiss message" onClick={()=>setNotice('')}/></div>}
  </div>;
}
createRoot(document.getElementById('root')).render(<App/>);
