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

// Arrow key navigation (← prev, → next)
// Skip when focus is inside an input/textarea to avoid interfering with typing.
document.addEventListener('keydown', e => {
  if (e.altKey || e.ctrlKey || e.metaKey) return;
  const tag = document.activeElement?.tagName;
  if (tag === 'INPUT' || tag === 'TEXTAREA') return;
  if (e.key === 'ArrowLeft') {
    const a = document.querySelector('#nav .prev');
    if (a) a.click();
  } else if (e.key === 'ArrowRight') {
    const a = document.querySelector('#nav .next');
    if (a) a.click();
  }
});

// Live reload via SSE
const es = new EventSource('/_reload');
es.addEventListener('reload', () => location.reload());

// Mermaid: replace ```mermaid blocks with div.mermaid
document.querySelectorAll('pre code.language-mermaid').forEach(block => {
  const div = document.createElement('div');
  div.className = 'mermaid';
  div.textContent = block.textContent;
  block.parentElement.replaceWith(div);
});
