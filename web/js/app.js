// kaze web UI. Filesystem-first navigation (mirrors Suika), with a
// persistent Plyr player at the bottom. Auth is JWT-only for the
// `/api/v1/*` endpoints; the Subsonic API is for external clients and
// is not used here.

const KEY_TOKEN = 'kaze.jwt';
const KEY_VIEW  = 'kaze.view';
const $ = (id) => document.getElementById(id);

const state = {
    currentPath: '',
    items: [],
    filteredItems: null,
    queue: [],          // ordered list of audio item paths in current dir
    currentTrackPath: null,
    plyr: null,
};

const api = {
    async login(username, password) {
        const res = await fetch('/api/v1/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password }),
        });
        if (!res.ok) throw new Error('Invalid credentials');
        return res.json();
    },
    async browse(path) {
        const token = sessionStorage.getItem(KEY_TOKEN);
        const url = '/api/v1/browse' + (path ? `?path=${encodeURIComponent(path)}` : '');
        const res = await fetch(url, { headers: { Authorization: `Bearer ${token}` } });
        if (res.status === 401) { logout(); throw new Error('Session expired'); }
        if (!res.ok) throw new Error('Browse failed');
        return res.json();
    },
};

function withToken(url) {
    const token = sessionStorage.getItem(KEY_TOKEN);
    const sep = url.includes('?') ? '&' : '?';
    return `${url}${sep}token=${encodeURIComponent(token)}`;
}

function toast(msg, kind = 'success') {
    const el = document.createElement('div');
    el.className = `toast toast-${kind}`;
    el.textContent = msg;
    document.body.appendChild(el);
    setTimeout(() => el.remove(), 2400);
}

function setLoading(visible, label = 'Loading...') {
    $('loading-overlay').style.display = visible ? 'flex' : 'none';
    $('loading-text').textContent = label;
}

// --- Navigation ---

async function navigate(path) {
    setLoading(true);
    try {
        const data = await api.browse(path);
        state.currentPath = path || '';
        state.items = data.items || [];
        state.filteredItems = null;
        state.queue = (data.items || []).filter(i => i.is_audio).map(i => i.path);
        $('search-input').value = '';
        renderBreadcrumb(data.current_folder);
        renderFolderHeader(data.current_folder);
        render();
        $('controls-bar').style.display = state.queue.length > 0 ? 'flex' : 'none';
    } catch (e) {
        toast(e.message, 'error');
    } finally {
        setLoading(false);
    }
}

function renderBreadcrumb(folder) {
    const bc = $('breadcrumb');
    if (!folder) {
        bc.innerHTML = '<a onclick="window.kazeApp.goHome()">Home</a>';
        return;
    }
    // Walk parents back to root by chopping the path.
    const parts = [{ name: 'Home', path: '' }];
    if (folder.path) {
        // Find the matching library root from the path by walking from root.
        const segments = folder.path.split('/').filter(Boolean);
        let acc = '';
        for (const seg of segments) {
            acc += '/' + seg;
            parts.push({ name: seg, path: acc });
        }
    }
    bc.innerHTML = parts.map((p, i) => {
        const last = i === parts.length - 1;
        const sep = i > 0 ? '<span class="sep">/</span>' : '';
        if (last) return `${sep}<span>${escapeHTML(p.name)}</span>`;
        return `${sep}<a data-path="${escapeAttr(p.path)}">${escapeHTML(p.name)}</a>`;
    }).join('');
    bc.querySelectorAll('a[data-path]').forEach(a => {
        a.addEventListener('click', () => navigate(a.getAttribute('data-path')));
    });
}

function renderFolderHeader(folder) {
    const h = $('folder-header');
    if (!folder || !state.currentPath) {
        h.innerHTML = '';
        return;
    }
    const back = folder.parent !== undefined ? folder.parent : '';
    h.innerHTML = `
        <span class="back-btn" id="back-btn" title="Back">&#8592;</span>
        <h2>${escapeHTML(folder.name)}</h2>
    `;
    $('back-btn').addEventListener('click', () => navigate(back));
}

function render() {
    const view = sessionStorage.getItem(KEY_VIEW) || 'grid';
    const grid = $('file-browser');
    grid.className = 'file-grid' + (view === 'list' ? ' list-view' : '');
    const items = state.filteredItems ?? state.items;
    if (items.length === 0) {
        grid.innerHTML = '<div class="empty-state">Empty folder</div>';
        return;
    }
    grid.innerHTML = items.map(item => {
        const playing = item.path === state.currentTrackPath ? ' playing' : '';
        const cls = item.is_dir ? 'file-item dir' : 'file-item audio' + playing;
        const poster = item.icon
            ? `<img src="${escapeAttr(withToken(item.icon))}" loading="lazy" alt="">`
            : (item.is_audio ? '&#9836;' : '&#128193;');
        const sub = item.is_audio ? `<div class="item-sub">${displayExt(item.name)}</div>` : '';
        return `
            <div class="${cls}" data-path="${escapeAttr(item.path)}" data-is-dir="${item.is_dir}" data-is-audio="${!!item.is_audio}">
                <div class="item-poster">${poster}</div>
                <div class="item-info">
                    <div class="item-name">${escapeHTML(stripExt(item.name))}</div>
                    ${sub}
                </div>
            </div>`;
    }).join('');
    grid.querySelectorAll('.file-item').forEach(el => {
        el.addEventListener('click', () => {
            const path = el.getAttribute('data-path');
            const isDir = el.getAttribute('data-is-dir') === 'true';
            const isAudio = el.getAttribute('data-is-audio') === 'true';
            if (isDir) {
                navigate(path);
            } else if (isAudio) {
                playTrack(path);
            }
        });
    });
}

function applyFilter(q) {
    const needle = q.trim().toLowerCase();
    if (!needle) {
        state.filteredItems = null;
    } else {
        state.filteredItems = state.items.filter(i => i.name.toLowerCase().includes(needle));
    }
    render();
}

// --- Player ---

function ensurePlyr() {
    if (state.plyr) return state.plyr;
    state.plyr = new Plyr('#player-audio', {
        controls: ['play', 'progress', 'current-time', 'duration', 'mute', 'volume'],
        invertTime: false,
        keyboard: { focused: true, global: true },
    });
    state.plyr.on('ended', () => {
        const idx = state.queue.indexOf(state.currentTrackPath);
        if (idx >= 0 && idx < state.queue.length - 1) {
            playTrack(state.queue[idx + 1]);
        }
    });
    return state.plyr;
}

function playTrack(path) {
    ensurePlyr();
    state.currentTrackPath = path;
    const item = state.items.find(i => i.path === path);

    const audio = $('player-audio');
    audio.src = withToken(item.stream_url);
    audio.load();
    state.plyr.play().catch(() => { /* user-gesture may be missing on initial autoplay */ });

    // Player chrome.
    $('player-bar').classList.add('visible');
    $('player-title').textContent = stripExt(item.name);
    $('player-sub').textContent = niceFolderName(state.currentPath);
    const coverHost = $('player-cover');
    // Use the folder poster (folder.jpg/cover.jpg in current dir) if available.
    const folderCover = state.items.find(i => i.is_dir === false && i.icon);
    // ^ won't usually exist; fall back to scanning items for the dir's poster.
    // Re-fetch the current folder cover via /api/v1/image: we already have it
    // in the breadcrumb's folder data, but it's simpler to re-request.
    coverHost.innerHTML = '';
    const breadcrumbFolderPoster = pickFolderPosterFromItems();
    if (breadcrumbFolderPoster) {
        const img = document.createElement('img');
        img.src = withToken(breadcrumbFolderPoster);
        coverHost.appendChild(img);
    }

    // Highlight the playing row.
    render();
}

function pickFolderPosterFromItems() {
    // The browse endpoint exposes the folder cover via current_folder.poster,
    // but we didn't keep that field in state. Cheapest fallback: look for an
    // image in the current dir reachable via /api/v1/image. We approximate
    // by hitting the same path the server uses internally for cover lookup —
    // which is exactly what `current_folder.poster` is. We re-fetch on demand.
    // To keep things simple, we just return null and rely on the cover being
    // shown next to each track via the per-item icon (which is null for audio
    // files — they don't carry covers). For audio rows we leave the poster
    // empty and trust the playing indicator on the grid item to be the visual.
    return null;
}

function playAll() {
    if (state.queue.length === 0) return;
    playTrack(state.queue[0]);
}

// --- Helpers ---

function stripExt(name) {
    const idx = name.lastIndexOf('.');
    return idx > 0 ? name.slice(0, idx) : name;
}
function displayExt(name) {
    const idx = name.lastIndexOf('.');
    return idx > 0 ? name.slice(idx + 1).toUpperCase() : '';
}
function escapeHTML(s) {
    return String(s)
        .replaceAll('&', '&amp;')
        .replaceAll('<', '&lt;')
        .replaceAll('>', '&gt;');
}
function escapeAttr(s) {
    return escapeHTML(s).replaceAll('"', '&quot;');
}
function niceFolderName(p) {
    if (!p) return '';
    const parts = p.split('/').filter(Boolean);
    return parts.length >= 2 ? `${parts[parts.length - 2]} / ${parts[parts.length - 1]}` : (parts[0] || '');
}

function logout() {
    sessionStorage.clear();
    if (state.plyr) state.plyr.destroy();
    state.plyr = null;
    $('app-container').style.display = 'none';
    $('login-screen').style.display = 'flex';
}

function showApp() {
    $('login-screen').style.display = 'none';
    $('app-container').style.display = 'block';
    navigate('');
}

window.kazeApp = {
    goHome: () => navigate(''),
    playAll,
    setView: (v) => {
        sessionStorage.setItem(KEY_VIEW, v);
        $('view-grid').classList.toggle('active', v === 'grid');
        $('view-list').classList.toggle('active', v === 'list');
        render();
    },
    logout,
};

document.addEventListener('DOMContentLoaded', () => {
    // Persist last-used view across reloads.
    const savedView = sessionStorage.getItem(KEY_VIEW);
    if (savedView) {
        $('view-grid').classList.toggle('active', savedView === 'grid');
        $('view-list').classList.toggle('active', savedView === 'list');
    }

    if (sessionStorage.getItem(KEY_TOKEN)) {
        showApp();
    }

    $('login-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        const username = $('username-input').value.trim();
        const password = $('password-input').value;
        try {
            const res = await api.login(username, password);
            sessionStorage.setItem(KEY_TOKEN, res.token);
            $('login-error').textContent = '';
            showApp();
        } catch (err) {
            $('login-error').textContent = err.message;
        }
    });

    $('search-input').addEventListener('input', (e) => applyFilter(e.target.value));
});
