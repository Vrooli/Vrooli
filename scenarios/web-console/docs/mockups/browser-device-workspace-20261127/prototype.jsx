import React, {useEffect, useRef, useState} from 'react';
import {createRoot} from 'react-dom/client';
import {Button} from '@vrooli/react-component-library/Button/2';
import {IconButton} from '@vrooli/react-component-library/IconButton';
import {ArrowDownRight, ArrowLeft, ArrowRight, ArrowUpRight, Check, ChevronDown, ChevronRight, Compass, Globe2, Hand, LockKeyhole, Maximize, MessageSquare, Moon, MoreHorizontal, Plus, RotateCw, ScanLine, Smartphone, Sun, X} from 'lucide-react';
import './prototype.css';

const designs = [
  ['a','Open the last target'],
  ['b','Work beside the agent'],
  ['c','Browse all targets']
];
const initial = [
  {id:'browser', name:'Default browser', kind:'Browser', detail:'Managed · Private profile', page:'Preview · Fieldnotes', state:'ready'},
  {id:'android', name:'Android test device', kind:'Emulated', detail:'Android 16 · Pixel 9', page:'Fieldnotes', state:'ready'},
  {id:'phone', name:'Desk Pixel', kind:'Physical', detail:'Android · Paired over Wi-Fi', page:'', state:'offline'}
];
const typeIcon = {Browser: Globe2, Emulated: Smartphone, Physical: Smartphone};
const names = ['Codex', 'Claude'];

function Icon({icon: Glyph, ...props}) { return <Glyph size={17} strokeWidth={1.75} aria-hidden="true" {...props}/>; }
function Action({icon, children, ...props}) { return <Button type="button" size="sm" {...props}>{icon && <Icon icon={icon}/>} {children}</Button>; }
function IconAction({icon, label, ...props}) { return <IconButton type="button" aria-label={label} title={label} size="sm" surface="ghost" {...props}><Icon icon={icon}/></IconButton>; }
function Badge({state, agent}) {
  const label = {ready:'Ready', agent:agent+' driving', human:'You’re driving', offline:'Offline', saved:'Saved'}[state];
  return <span className={'pill '+state}><i/>{label}</span>;
}
function TargetGlyph({target}) {
  return <span className={'targetGlyph '+target.kind.toLowerCase()}><Icon icon={typeIcon[target.kind]}/>{target.kind==='Emulated' && <i/>}</span>;
}
function TargetRows({targets, selected, onSelect}) {
  return <div className="targetRows">{targets.map(t=><button key={t.id} type="button" className="targetRow" onClick={()=>onSelect(t.id)} aria-current={t.id===selected?'true':undefined}>
    <TargetGlyph target={t}/><span className="targetText"><strong>{t.name}</strong><small>{t.kind} · {t.kind==='Browser'?t.page:t.detail.split(' · ')[0]}</small></span>
    <Badge state={t.state} agent={t.agent}/><Icon icon={t.id===selected?Check:ChevronRight}/>
  </button>)}</div>;
}

const seedNotes=['Checkout flow','Mobile spacing','Release checklist'];
function Fieldnotes({phone, notes, canInput, toggleNote}) {
  const [selected, setSelected] = useState(0);
  return <div className={'fieldnotes '+(phone?'deviceCanvas':'')}>
    <div className="fieldTop"><span className="fieldLogo"><span>✳</span> Fieldnotes</span><span className="fieldProject">Storefront QA</span></div>
    <div className="fieldBody">
      <div className="fieldRail"><b>Workspace</b><span>Overview</span><span className="fieldNavActive">Test checklist</span><span>Release notes</span></div>
      <div className="fieldContent">
        <div className="fieldCrumb">Storefront <ChevronRight size={11}/> QA</div>
        <h2>Test checklist</h2>
        <p>Verify the preview before release.</p>
        <div className="fieldList">{seedNotes.map((note,index)=><button type="button" key={note} disabled={!canInput} onClick={()=>{setSelected(index); toggleNote(index)}} className={selected===index?'fieldSelected':''}><span className={'fieldCheck '+(notes.includes(index)?'checked':'')}>{notes.includes(index)&&<Check size={12}/>}</span><span>{note}</span><ChevronRight size={13}/></button>)}</div>
      </div>
    </div>
  </div>;
}
function Conversation({agent, state, onShare, onClose, inSheet=false}) {
  return <section className="conversation">
    <header><div className="conversationTitle"><Icon icon={MessageSquare}/><strong>Storefront QA</strong></div>{inSheet&&<IconAction icon={X} label="Close conversation" onClick={onClose}/>}</header>
    <div className="conversationScroll">
      <div className="speaker"><span className="sender humanSender">M</span><strong>You</strong></div>
      <div className="message you">Check the checkout preview. Keep it open so I can verify the layout.</div>
      <div className="speaker"><span className="sender agentSender">✳</span><strong>{agent || 'Codex'}</strong></div>
      <div className="message">I opened the preview. The checkout path is on the checklist. You can take over to test it yourself.</div>
      <div className="sharedMessage"><Icon icon={Globe2}/><div><b>Default browser</b><small>{state==='agent'?(agent||'Codex')+' driving':state==='human'?'You’re driving':'Ready to share'}</small></div><Icon icon={ArrowUpRight}/></div>
    </div>
    <div className="conversationFooter"><Action icon={ArrowUpRight} variant="secondary" onClick={onShare}>Share a target</Action></div>
  </section>;
}
function Dialog({title, children, close}) {
  const ref = useRef();
  useEffect(()=>{ const node=ref.current; node.showModal(); return ()=>node.close(); },[]);
  return <dialog ref={ref} aria-label={title} onCancel={e=>{e.preventDefault();close();}}>
    <div className="dialogHead"><h2>{title}</h2><IconAction icon={X} label="Close" onClick={close}/></div>{children}
  </dialog>;
}

function App() {
  const params = new URLSearchParams(location.search);
  const [design,setDesign] = useState(designs.some(([id])=>id===params.get('view'))?params.get('view'):'a');
  const [theme,setTheme] = useState(params.get('theme')||'dark');
  const [targets,setTargets] = useState(()=>initial.map(t=>t.id==='browser'&&design==='b'?{...t,state:'agent',agent:'Codex'}:{...t}));
  const [id,setID] = useState('browser');
  const [library,setLibrary] = useState(design==='c');
  const [modal,setModal] = useState(null);
  const [notice,setNotice] = useState('');
  const [notes,setNotes] = useState({browser:[0]});
  const [address,setAddress] = useState('preview.local/checklist');
  const [chromeHidden,setChromeHidden] = useState(false);
  const [rotated,setRotated] = useState(false);
  const target=targets.find(t=>t.id===id);
  const agent=target.agent;
  const activeAgent=target.state==='agent';
  const driving=target.state==='human';
  const phone=target.kind!=='Browser';
  useEffect(()=>{document.documentElement.dataset.theme=theme},[theme]);
  const update=patch=>setTargets(prev=>prev.map(t=>t.id===id?{...t,...patch}:t));
  function switchDesign(next) {
    setDesign(next);
    setLibrary(next==='c');
    setModal(null);
  }
  function openTarget(next) {
    setID(next); setLibrary(false); setModal(null); setChromeHidden(false); setRotated(false);
    setAddress(targets.find(t=>t.id===next)?.url||'preview.local/checklist');
  }
  function share(withAgent) {
    update({state:'agent', agent:withAgent});
    setModal(null);
    setNotice(withAgent+' is now driving '+target.name+'.');
  }
  function takeControl() {
    update({state:'human'});
    setNotice(agent?'Control passed from '+agent+' to you.':'You have control.');
  }
  function releaseControl() {
    update({state:agent?'agent':'ready'});
    setNotice(agent?'Control returned to '+agent+'.':'Target is ready to share.');
  }
  function finishAgent() {
    update({state:'ready',agent:undefined});
    setNotice(agent+' disconnected. The target is ready with its work intact.');
    setModal(null);
  }
  function add(kind) {
    if(kind==='Physical'){setModal('pair');return;}
    const key='target-'+(targets.length+1);
    const name=kind==='Browser'?'Browser '+(targets.filter(t=>t.kind==='Browser').length+1):'Android emulator '+(targets.filter(t=>t.kind==='Emulated').length+1);
    setTargets(prev=>[...prev,{id:key,name,kind,detail:kind==='Browser'?'Managed · Isolated profile':'Android 16 · Pixel 9',page:'Start page',url:'',state:'ready'}]);
    setID(key); setAddress(''); setLibrary(false); setModal(null); setChromeHidden(false); setRotated(false);
    setNotice(name+' created.');
  }
  function navigate(event) {
    event.preventDefault();
    if(driving&&address.trim()){
      update({url:address.trim(),page:'Preview · Fieldnotes'});
      setNotice('Opened '+address.trim()+'.');
    }
  }
  function toggleNote(index) {
    const current=notes[id]||[];
    setNotes({...notes,[id]:current.includes(index)?current.filter(n=>n!==index):[...current,index]});
  }
  const controlAction=()=>{
    if(target.state==='offline') return <Action variant="secondary" icon={RotateCw} onClick={()=>setNotice('Desk Pixel is still offline. Bring it online to reconnect.')}>Check connection</Action>;
    if(target.state==='saved') return <Action variant="primary" icon={RotateCw} onClick={()=>{update({state:'ready'});setNotice('Browser reopened.')}}>Resume browser</Action>;
    if(driving) return <Action variant="primary" icon={ArrowDownRight} onClick={releaseControl}>{agent?'Return to '+agent:'Release control'}</Action>;
    return <Action variant="primary" icon={Hand} onClick={takeControl}>{activeAgent?'Take control':phone?'Control device':'Use browser'}</Action>;
  };
  const targetActions = <>
    {target.state!=='offline'&&target.state!=='saved'&&<Action variant="secondary" icon={ArrowUpRight} onClick={()=>setModal('share')}>{activeAgent?'Handoff':'Share with agent'}</Action>}
    {controlAction()}
  </>;
  const options=<>
    {target.state==='agent'&&<Action variant="secondary" onClick={finishAgent}>Simulate agent finishing</Action>}
    {target.kind==='Browser'&&target.state==='ready'&&<Action variant="secondary" onClick={()=>{update({state:'saved'});setModal(null);setNotice('Browser saved.')}}>Save and sleep browser</Action>}
    <Action variant="secondary" onClick={()=>{setModal(null);setLibrary(true)}}>Show all targets</Action>
  </>;
  return <div className="demo">
    <div className="studybar">
      <strong>Design preview</strong>
      <nav aria-label="Concepts">{designs.map(([key,title])=><button key={key} type="button" onClick={()=>switchDesign(key)} aria-pressed={design===key}><span>{key.toUpperCase()}</span><span className="optionLabel">{title}</span></button>)}</nav>
      <IconAction icon={theme==='dark'?Sun:Moon} label={theme==='dark'?'Use light theme':'Use dark theme'} onClick={()=>setTheme(theme==='dark'?'light':'dark')}/>
    </div>
    <div className={'shell '+(design==='b'&&!library?'paired':'')} data-design={design}>
      <div className="appbar">
        <div className="brand"><span><Icon icon={Compass}/></span>Aquila</div>
        <div className="appTabs"><span className="workspaceName">Workspace</span><button type="button" className="appActive" onClick={()=>{if(design==='c')setLibrary(true);else if(library)openTarget(id)}}><Icon icon={Globe2}/> Browser &amp; devices</button></div>
        <div className="appTools"><span className="avatar" title="You">M</span></div>
      </div>
      {library?<main className="library">
        <div className="libraryHeading"><div><h1>Browser &amp; devices</h1><p>Pick up where you left off, or open a new target.</p></div><Action icon={Plus} variant="primary" onClick={()=>setModal('add')}>Add target</Action></div>
        <div className="targetGrid">{targets.map(t=><button type="button" className="targetCard" key={t.id} onClick={()=>openTarget(t.id)}>
          <div className={'miniStage '+t.kind.toLowerCase()}><TargetGlyph target={t}/></div>
          <div className="cardBody"><Badge state={t.state} agent={t.agent}/><h2>{t.name}</h2><p>{t.detail}</p><span className="cardOpen">{t.state==='offline'?'View status':'Open'} <Icon icon={ArrowRight}/></span></div>
        </button>)}</div>
      </main>:<div className="work">
        {design==='b'&&<div className="conversationDesktop"><Conversation agent={agent} state={target.state} onShare={()=>setModal('share')}/></div>}
        <main className="viewer">
          <section className="viewerHeader" aria-label="Current target">
            <div className="targetHeading">
              {design==='c'&&<IconAction icon={ArrowLeft} label="All targets" onClick={()=>setLibrary(true)}/>}
              <button type="button" className="targetSelect" onClick={()=>setModal('targets')}><TargetGlyph target={target}/><span><strong>{target.name}</strong><small>{target.kind==='Browser'?'Browser':target.kind+' device'}</small></span><Icon icon={ChevronDown}/></button>
              <Badge state={target.state} agent={agent}/>
            </div>
            <div className="viewerActions">{targetActions}</div>
            <div className="headerMore"><IconAction icon={MoreHorizontal} label="Target options" onClick={()=>setModal('options')}/></div>
          </section>
          <section className={'viewerStage '+(phone?'phoneStage':'')+(rotated?' rotated':'')} aria-label={phone?'Device preview':'Browser preview'}>
            {target.state==='offline'?<div className="blank"><TargetGlyph target={target}/><h2>{target.name} is offline</h2><p>Bring the device online to reconnect.</p></div>
            :target.state==='saved'?<div className="blank"><TargetGlyph target={target}/><h2>Browser is asleep</h2><p>Resume to restore its saved tabs and profile.</p></div>
            :<div className={'surface '+(phone?'phoneSurface':'')}>
              {!phone&&!chromeHidden&&<>
                <div className="browserTabBar"><span className="browserTab"><Icon icon={Globe2}/> {target.page}</span></div>
                <form onSubmit={navigate} className="addressBar">
                  <div className="address"><Icon icon={LockKeyhole}/><input aria-label="Browser address" value={address} readOnly={!driving} onChange={e=>setAddress(e.target.value)} placeholder="Enter an address"/></div>
                  <IconAction icon={Maximize} label="Focus the browser" onClick={()=>setChromeHidden(true)}/>
                </form>
              </>}
              {!phone&&chromeHidden&&<div className="addressBar focusBar"><Action variant="secondary" onClick={()=>setChromeHidden(false)}>Show browser controls</Action></div>}
              {target.page==='Start page'?<div className="blank start"><Icon icon={Globe2} size={36}/><h2>Open a page</h2><p>Take control to enter an address.</p></div>:<Fieldnotes key={id} phone={phone} notes={notes[id]||[]} canInput={driving} toggleNote={toggleNote}/>}
              {phone&&<div className="deviceHome"><span/></div>}
            </div>}
          </section>
          <div className="viewerFoot">
            <span>{activeAgent?'Watching live':driving?'Input active':target.state==='ready'?'Available to use or share':target.state==='offline'?'Device connection required':'Workspace saved'}</span>
            {phone&&target.state!=='offline'&&<IconAction icon={ScanLine} label="Rotate device" onClick={()=>setRotated(prev=>!prev)}/>}
            {design==='b'&&<button className="conversationMobile" type="button" onClick={()=>setModal('conversation')}><Icon icon={MessageSquare}/> Conversation</button>}
          </div>
          <div className="mobileActions">{targetActions}</div>
        </main>
      </div>}
    </div>
    {modal==='targets'&&<Dialog title="Switch target" close={()=>setModal(null)}><TargetRows targets={targets} selected={id} onSelect={openTarget}/><div className="dialogFoot"><Action variant="secondary" onClick={()=>{setModal(null);setLibrary(true)}}>All targets</Action><Action icon={Plus} variant="primary" onClick={()=>setModal('add')}>Add target</Action></div></Dialog>}
    {modal==='add'&&<Dialog title="Add target" close={()=>setModal(null)}><div className="choiceList">
      {[['Browser','New browser','Start with its own tabs and profile.'],['Emulated','New emulator','Start a virtual Android device.'],['Physical','Pair a device','Connect supported hardware.']].map(([kind,label,desc])=><button className="targetRow" key={kind} type="button" onClick={()=>add(kind)}><TargetGlyph target={{kind}}/><span className="targetText"><strong>{label}</strong><small>{desc}</small></span><Icon icon={ChevronRight}/></button>)}
    </div></Dialog>}
    {modal==='pair'&&<Dialog title="Pair a device" close={()=>setModal(null)}><p className="dialogCopy">Aquila would open Device Control’s pairing flow here. This preview uses the paired Desk Pixel as its example.</p><div className="dialogFoot"><Action variant="secondary" onClick={()=>setModal('add')}>Back</Action><Action variant="primary" onClick={()=>openTarget('phone')}>View paired device</Action></div></Dialog>}
    {modal==='share'&&<Dialog title={activeAgent?'Hand off '+target.name:'Share '+target.name} close={()=>setModal(null)}>
      <p className="dialogCopy">{driving?'Sharing transfers control. The agent receives access to this target only.':'Select an agent to grant access to this target. One driver can send input at a time.'}</p>
      <div className="choiceList">{names.filter(n=>!activeAgent||n!==agent).map(n=><button className="targetRow" key={n} type="button" onClick={()=>share(n)}><span className="sender agentSender">✳</span><span className="targetText"><strong>{n}</strong><small>Grant control</small></span><Icon icon={ArrowRight}/></button>)}</div>
      {activeAgent&&<div className="dialogFoot"><Action variant="secondary" onClick={finishAgent}>Simulate {agent} finishing</Action></div>}
    </Dialog>}
    {modal==='options'&&<Dialog title={target.name} close={()=>setModal(null)}><div className="menuOptions">{options}</div></Dialog>}
    {modal==='conversation'&&<Dialog title="Conversation" close={()=>setModal(null)}><Conversation agent={agent} state={target.state} onShare={()=>setModal('share')}/></Dialog>}
    <div className="srOnly" role="status">{notice}</div>
  </div>;
}

createRoot(document.getElementById('root')).render(<App/>);
