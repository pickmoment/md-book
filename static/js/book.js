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

// AI Chat Panel
(function () {
  const panel = document.getElementById('ai-panel');
  const tooltip = document.getElementById('ai-tooltip');
  const closeBtn = document.getElementById('ai-panel-close');
  const openBtn = document.getElementById('ai-open-btn');
  const input = document.getElementById('ai-input');
  const sendBtn = document.getElementById('ai-send');
  const messagesEl = document.getElementById('ai-messages');
  const contextBox = document.getElementById('ai-context-box');
  const resizeHandle = document.getElementById('ai-resize-handle');

  const AI_WIDTH_KEY = 'md-book-ai-width';
  const savedWidth = parseInt(localStorage.getItem(AI_WIDTH_KEY));
  if (savedWidth) panel.style.width = savedWidth + 'px';

  let aiContext = '';
  let aiMessages = [];
  let pendingSelection = '';

  function panelWidth() { return panel.offsetWidth; }

  function openPanel(selectedText) {
    if (selectedText) {
      aiContext = selectedText;
      contextBox.textContent = selectedText;
      contextBox.classList.add('has-context');
    }
    panel.classList.add('open');
    openBtn.style.display = 'none';
    if (window.innerWidth > 768) {
      document.getElementById('main').style.marginRight = panelWidth() + 'px';
    }
    hideTooltip();
    setTimeout(() => input.focus(), 50);
    ensureMarked();
  }

  function closePanel() {
    panel.classList.remove('open');
    openBtn.style.display = '';
    document.getElementById('main').style.marginRight = '';
  }

  function hideTooltip() {
    tooltip.style.display = 'none';
    pendingSelection = '';
  }

  closeBtn.addEventListener('click', closePanel);
  openBtn.addEventListener('click', () => openPanel(''));

  // Text selection → Ask AI tooltip
  document.addEventListener('mouseup', () => {
    const sel = window.getSelection();
    if (!sel || sel.isCollapsed) { hideTooltip(); return; }
    const text = sel.toString().trim();
    if (!text) { hideTooltip(); return; }
    const content = document.getElementById('content');
    if (!content) { hideTooltip(); return; }
    const range = sel.getRangeAt(0);
    if (!content.contains(range.commonAncestorContainer)) { hideTooltip(); return; }

    const rect = range.getBoundingClientRect();
    const top = rect.top > 42 ? rect.top - 36 : rect.bottom + 6;
    const left = Math.min(Math.max(4, rect.left), window.innerWidth - 110);
    tooltip.style.top = top + 'px';
    tooltip.style.left = left + 'px';
    tooltip.style.display = 'block';
    pendingSelection = text;
  });

  document.addEventListener('mousedown', e => {
    if (e.target !== tooltip) hideTooltip();
  });

  tooltip.addEventListener('mousedown', e => {
    e.preventDefault();
    openPanel(pendingSelection);
  });

  // Resize handle
  let resizing = false, resizeStartX = 0, resizeStartW = 0;

  resizeHandle.addEventListener('mousedown', e => {
    e.preventDefault();
    resizing = true;
    resizeStartX = e.clientX;
    resizeStartW = panelWidth();
    resizeHandle.classList.add('dragging');
    document.body.style.cursor = 'ew-resize';
    document.body.style.userSelect = 'none';
  });

  document.addEventListener('mousemove', e => {
    if (!resizing) return;
    const delta = resizeStartX - e.clientX;
    const newW = Math.max(280, Math.min(resizeStartW + delta, Math.floor(window.innerWidth * 0.75)));
    panel.style.width = newW + 'px';
    if (document.getElementById('main').style.marginRight) {
      document.getElementById('main').style.marginRight = newW + 'px';
    }
  });

  document.addEventListener('mouseup', () => {
    if (!resizing) return;
    resizing = false;
    resizeHandle.classList.remove('dragging');
    document.body.style.cursor = '';
    document.body.style.userSelect = '';
    localStorage.setItem(AI_WIDTH_KEY, panelWidth());
  });

  // Lazy-load marked.js for markdown rendering
  let _markedPromise = null;
  function ensureMarked() {
    if (_markedPromise) return _markedPromise;
    _markedPromise = new Promise((resolve, reject) => {
      if (window.marked) { resolve(); return; }
      const s = document.createElement('script');
      s.src = 'https://cdn.jsdelivr.net/npm/marked/marked.min.js';
      s.onload = resolve;
      s.onerror = () => { _markedPromise = null; reject(new Error('marked load failed')); };
      document.head.appendChild(s);
    });
    return _markedPromise;
  }

  // Messages
  function appendMessage(role, content) {
    const div = document.createElement('div');
    div.className = 'ai-msg ' + role;
    div.textContent = content;
    messagesEl.appendChild(div);
    messagesEl.scrollTop = messagesEl.scrollHeight;
    return div;
  }

  async function sendMessage() {
    const q = input.value.trim();
    if (!q || sendBtn.disabled) return;
    input.value = '';
    sendBtn.disabled = true;

    appendMessage('user', q);
    aiMessages.push({ role: 'user', content: q });

    const loading = appendMessage('assistant', '생각 중...');
    loading.classList.add('loading');

    try {
      const [res] = await Promise.all([
        fetch('/_ask', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ messages: aiMessages, context: aiContext }),
        }),
        ensureMarked(),
      ]);
      const data = await res.json();
      loading.remove();
      if (data.error) {
        const errMsg = appendMessage('assistant', '오류: ' + data.error);
        errMsg.style.color = '#c0392b';
        aiMessages.pop();
      } else {
        const msgEl = document.createElement('div');
        msgEl.className = 'ai-msg assistant';
        msgEl.innerHTML = marked.parse(data.answer);
        messagesEl.appendChild(msgEl);
        messagesEl.scrollTop = messagesEl.scrollHeight;
        aiMessages.push({ role: 'assistant', content: data.answer });
      }
    } catch (err) {
      loading.remove();
      const errMsg = appendMessage('assistant', '오류: ' + err.message);
      errMsg.style.color = '#c0392b';
      aiMessages.pop();
    } finally {
      sendBtn.disabled = false;
      input.focus();
    }
  }

  sendBtn.addEventListener('click', sendMessage);
  input.addEventListener('keydown', e => {
    if (e.key === 'Enter' && !e.shiftKey && !e.isComposing) { e.preventDefault(); sendMessage(); }
  });

  document.addEventListener('keydown', e => {
    if (e.key === 'Escape' && panel.classList.contains('open')) closePanel();
  });
})();

// EPUB export modal
(function () {
  const modal = document.getElementById('epub-modal');
  const openBtn = document.getElementById('epub-open-modal');
  const cancelBtn = document.getElementById('epub-cancel');
  const downloadBtn = document.getElementById('epub-download');
  const titleInput = document.getElementById('epub-title');
  const authorInput = document.getElementById('epub-author');
  if (!modal || !openBtn) return;

  function openModal() {
    modal.hidden = false;
    titleInput.focus();
    titleInput.select();
  }

  function closeModal() {
    modal.hidden = true;
  }

  function startDownload() {
    const title = titleInput.value.trim() || titleInput.placeholder;
    const author = authorInput.value.trim();
    const params = new URLSearchParams({ title });
    if (author) params.set('author', author);
    const a = document.createElement('a');
    a.href = '/_export/epub?' + params.toString();
    a.download = '';
    document.body.appendChild(a);
    a.click();
    a.remove();
    closeModal();
  }

  openBtn.addEventListener('click', openModal);
  cancelBtn.addEventListener('click', closeModal);
  downloadBtn.addEventListener('click', startDownload);

  modal.addEventListener('click', e => { if (e.target === modal) closeModal(); });

  document.addEventListener('keydown', e => {
    if (e.key === 'Escape' && !modal.hidden) closeModal();
    if (e.key === 'Enter' && !modal.hidden) {
      const tag = document.activeElement?.tagName;
      if (tag !== 'BUTTON') startDownload();
    }
  });
})();

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
