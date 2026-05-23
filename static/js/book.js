// Sidebar toggle (mobile)
const toggle = document.getElementById('sidebar-toggle');
const sidebar = document.getElementById('sidebar');
if (toggle && sidebar) {
  toggle.addEventListener('click', () => sidebar.classList.toggle('open'));
}

// Font size control
const FONT_KEY = 'md-book-font-size';
const FONT_MIN = 14, FONT_MAX = 24, FONT_STEP = 1, FONT_DEFAULT = 18;

function applyFontSize(px) {
  document.documentElement.style.fontSize = px + 'px';
  const label = document.getElementById('font-size-label');
  if (label) label.textContent = px + 'px';
  document.getElementById('font-decrease').disabled = px <= FONT_MIN;
  document.getElementById('font-increase').disabled = px >= FONT_MAX;
}

function currentFontSize() {
  return parseInt(localStorage.getItem(FONT_KEY)) || FONT_DEFAULT;
}

applyFontSize(currentFontSize());

document.getElementById('font-decrease').addEventListener('click', () => {
  const next = Math.max(FONT_MIN, currentFontSize() - FONT_STEP);
  localStorage.setItem(FONT_KEY, next);
  applyFontSize(next);
});

document.getElementById('font-increase').addEventListener('click', () => {
  const next = Math.min(FONT_MAX, currentFontSize() + FONT_STEP);
  localStorage.setItem(FONT_KEY, next);
  applyFontSize(next);
});

// Content width control
const WIDTH_KEY = 'md-book-width';
const WIDTH_DEFAULT = '38rem';

function applyWidth(value) {
  document.documentElement.style.setProperty('--content-width', value);
  document.querySelectorAll('#width-controls button').forEach(btn => {
    btn.classList.toggle('active', btn.dataset.width === value);
  });
}

const savedWidth = localStorage.getItem(WIDTH_KEY) || WIDTH_DEFAULT;
applyWidth(savedWidth);

document.querySelectorAll('#width-controls button').forEach(btn => {
  btn.addEventListener('click', () => {
    localStorage.setItem(WIDTH_KEY, btn.dataset.width);
    applyWidth(btn.dataset.width);
  });
});

// SPA-style navigation: fetch only the content fragment, swap it in place,
// then update the URL with history.pushState. This avoids a full page reload
// so the browser loading indicator never fires on arrow-key navigation.
let _navPending = false;

async function navigateTo(url) {
  if (_navPending) return;
  _navPending = true;
  try {
    const res = await fetch(url);
    if (!res.ok) { location.href = url; return; }
    const html = await res.text();
    const doc = new DOMParser().parseFromString(html, 'text/html');

    document.getElementById('content').innerHTML = doc.getElementById('content').innerHTML;
    document.getElementById('nav').innerHTML = doc.getElementById('nav').innerHTML;
    document.title = doc.title;

    const newPath = new URL(url, location.origin).pathname;
    document.querySelectorAll('#sidebar a[href]').forEach(a => {
      a.classList.toggle('active', new URL(a.href, location.origin).pathname === newPath);
    });

    history.pushState({ url }, '', url);
    window.scrollTo(0, 0);
    initMermaid();
  } catch (_) {
    location.href = url;
  } finally {
    _navPending = false;
  }
}

history.replaceState({ url: location.href }, '');
window.addEventListener('popstate', e => {
  if (e.state?.url) navigateTo(e.state.url);
});

// Sidebar TOC link navigation
document.getElementById('sidebar').addEventListener('click', e => {
  const a = e.target.closest('a[href]');
  if (!a || a.id === 'sidebar-title') return;
  if (new URL(a.href).origin !== location.origin) return;
  e.preventDefault();
  navigateTo(a.href);
});

// Arrow key navigation (← prev, → next)
document.addEventListener('keydown', e => {
  if (e.altKey || e.ctrlKey || e.metaKey) return;
  const tag = document.activeElement?.tagName;
  if (tag === 'INPUT' || tag === 'TEXTAREA') return;
  if (e.key === 'ArrowLeft') {
    const a = document.querySelector('#nav .prev');
    if (a) { e.preventDefault(); navigateTo(a.href); }
  } else if (e.key === 'ArrowRight') {
    const a = document.querySelector('#nav .next');
    if (a) { e.preventDefault(); navigateTo(a.href); }
  }
});

// Live reload — started after load so the open SSE connection does not keep
// the browser loading indicator spinning while the page is still settling.
window.addEventListener('load', () => {
  const es = new EventSource('/_reload');
  es.addEventListener('reload', () => location.reload());
});

// Mermaid: load from CDN only on pages that actually contain diagrams.
function initMermaid() {
  const blocks = document.querySelectorAll('pre code.language-mermaid');
  if (blocks.length === 0) return;
  blocks.forEach(block => {
    const div = document.createElement('div');
    div.className = 'mermaid';
    div.textContent = block.textContent;
    block.parentElement.replaceWith(div);
  });
  if (window.mermaid) {
    mermaid.run();
  } else {
    const s = document.createElement('script');
    s.src = 'https://cdn.jsdelivr.net/npm/mermaid/dist/mermaid.min.js';
    s.onload = () => mermaid.initialize({ startOnLoad: false }) || mermaid.run();
    document.head.appendChild(s);
  }
}
initMermaid();
