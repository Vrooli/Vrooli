import React,{useEffect,useRef} from 'react';
import {Circle, Clock3, Radio} from 'lucide-react';
import collectionWide from './history-collection-wide.webp';
import collectionNarrow from './history-collection-narrow.webp';
import productWide from './history-product-wide.webp';
import productNarrow from './history-product-narrow.webp';
import shippingWide from './history-shipping-wide.webp';
import shippingNarrow from './history-shipping-narrow.webp';
import cartWide from './history-cart-wide.webp';
import cartNarrow from './history-cart-narrow.webp';

const captures={
  collection:[collectionWide,collectionNarrow],
  product:[productWide,productNarrow],
  shipping:[shippingWide,shippingNarrow],
  cart:[cartWide,cartNarrow]
};

export const initialActivity=[
  {id:'first',page:'collection',action:'Opened storefront',agent:'Claude',time:'9:41 AM'},
  {id:'second',page:'product',action:'Opened product',agent:'Claude',time:'9:42 AM'},
  {id:'third',page:'shipping',action:'Checked shipping',agent:'Codex',time:'9:48 AM'},
  {id:'fourth',page:'cart',action:'Checked shopping bag',agent:'Codex',time:'9:50 AM'}
];

export function HistoryFrame({item,device}) {
  const [wide,narrow]=captures[item.page];
  return <picture className={'historyFrame '+(device?'phoneFrame':'')}>
    <source media={device?'all':'(max-width: 650px)'} srcSet={narrow}/>
    <img src={wide} alt={'Saved screenshot: '+item.action}/>
  </picture>;
}

export function ActivityTimeline({entries, selected, onSelect, agent, live}) {
  const track=useRef(null);
  const scroll=useRef(null);
  const position=selected?entries.findIndex(entry=>entry.id===selected.id):entries.length;
  useEffect(()=>{
    if(!track.current) return;
    if(!selected) { track.current.scrollLeft=track.current.scrollWidth;return; }
    const node=scroll.current;
    if(node&&node.offsetLeft<track.current.scrollLeft) track.current.scrollLeft=node.offsetLeft;
    if(node&&node.offsetLeft+node.offsetWidth>track.current.scrollLeft+track.current.clientWidth) {
      track.current.scrollLeft=node.offsetLeft+node.offsetWidth-track.current.clientWidth;
    }
  },[selected,entries.length]);
  return <section className="activityTimeline" aria-label="Agent activity timeline">
    <div className="activityTop">
      <div className="activityTitle"><Clock3 size={15} strokeWidth={1.8} aria-hidden="true"/><h2>Agent activity</h2></div>
      {agent&&live&&<span className="activityRecording"><Radio size={14} aria-hidden="true"/> <span>Capturing screenshots</span></span>}
    </div>
    {entries.length>0?<div className="activityContent">
      <div className="scrubRow">
        <input type="range" aria-label="Review agent activity" aria-valuetext={selected?selected.agent+' · '+selected.action+' · '+selected.time:live?'Live':'Current'} min="0" max={entries.length} value={position} onChange={event=>{const index=Number(event.target.value);onSelect(entries[index]||null)}}/>
      </div>
      <div className="filmstrip">
        <div className="frames" ref={track}>
          {entries.map(entry=><button type="button" className="frame" key={entry.id} ref={selected?.id===entry.id?scroll:null} aria-pressed={selected?.id===entry.id} onClick={()=>onSelect(entry)}>
            <img src={captures[entry.page][0]} alt=""/>
            <span><b>{entry.action}</b><small>{entry.agent} · {entry.time}</small></span>
          </button>)}
        </div>
        <button type="button" className="timelineLive" aria-pressed={!selected} onClick={()=>onSelect(null)} aria-label={live?'Live, current view':'Current target view'}>
          <Circle size={9} fill="currentColor" aria-hidden="true"/> {live?'Live':'Current'}
        </button>
      </div>
    </div>:<div className="activityEmpty">Screenshots appear here when an agent works on this target.</div>}
  </section>;
}
