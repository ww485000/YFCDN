const app = document.getElementById('app');
const apiBase = '';

const defaultLang = 'zh-CN';
let lang = localStorage.getItem('yfcdn_lang') || defaultLang;
let token = localStorage.getItem('yfcdn_token') || '';
let current = localStorage.getItem('yfcdn_view') || 'dashboard';
let cache = { dashboard: null, tenants: [], operators: [], nodes: [], domains: [], plans: [], keys: [] };

const viewIds = ['dashboard', 'tenants', 'operators', 'nodes', 'domains', 'plans', 'keys', 'docs'];

const i18n = {
  'zh-CN': {
    'app.title': 'YFCDN 控制台',
    'app.intro': '轻量 CDN 控制面板，支持多租户运营、边缘节点、域名管理、API Key 与节点配置下发。',
    'app.footer': 'YFCDN 原创轻量 UI 模板，支持运营商面板和 API 自动化。',
    'auth.username': '用户名',
    'auth.password': '密码',
    'auth.login': '登录',
    'auth.defaultPassword': '默认密码来自 YFCDN_ADMIN_PASSWORD，未设置时为 admin123。公网部署前请务必修改。',
    'auth.invalid': '登录失败',
    'auth.expired': '登录已过期，请重新登录。',
    'top.refresh': '刷新',
    'top.logout': '退出',
    'top.language': '语言',
    'common.create': '创建',
    'common.delete': '删除',
    'common.copy': '复制',
    'common.copied': '已复制',
    'common.copyManual': '自动复制失败，请手动复制：',
    'common.noData': '暂无数据',
    'common.actions': '操作',
    'common.status': '状态',
    'common.created': '创建时间',
    'common.active': '启用',
    'common.paused': '暂停',
    'common.offline': '离线',
    'common.pending': '待处理',
    'common.name': '名称',
    'common.contact': '联系方式',
    'common.remark': '备注',
    'common.all': '全部',
    'common.none': '无',
    'common.yes': '是',
    'common.no': '否',
    'common.confirmDelete': '确认删除或吊销该项目？',
    'common.saved': '操作成功',
    'common.loading': '加载中...',
    'view.dashboard': '仪表盘',
    'view.tenants': '客户/租户',
    'view.operators': '运营商',
    'view.nodes': '节点管理',
    'view.domains': '域名管理',
    'view.plans': '套餐管理',
    'view.keys': 'API Key',
    'view.docs': '节点安装',
    'subtitle.dashboard': '系统概览、在线节点和最近事件。',
    'subtitle.tenants': '管理客户、配额和租户状态。',
    'subtitle.operators': '管理运营商、代理商和 API 调用能力。',
    'subtitle.nodes': '注册边缘节点，查看心跳、负载和安装命令。',
    'subtitle.domains': '绑定客户域名、源站和分配节点。',
    'subtitle.plans': '创建可销售的流量或带宽套餐。',
    'subtitle.keys': '签发 API Key 给自动化系统、运营商或租户。',
    'subtitle.docs': '一键安装边缘 Agent，支持主控和节点同机部署。',
    'metric.tenants': '租户数',
    'metric.onlineNodes': '在线节点',
    'metric.activeDomains': '启用域名',
    'metric.traffic': '累计流量',
    'dashboard.events': '最近事件',
    'table.time': '时间',
    'table.type': '类型',
    'table.actor': '操作者',
    'table.message': '消息',
    'tenant.create': '创建租户',
    'tenant.list': '租户列表',
    'tenant.quota': '流量配额 GB',
    'operator.create': '创建运营商',
    'operator.list': '运营商列表',
    'operator.apiEnabled': '允许 API',
    'operator.tenantIds': '租户 ID，多个用英文逗号分隔',
    'node.create': '创建节点',
    'node.list': '节点列表',
    'node.ip': 'IP 地址',
    'node.region': '区域',
    'node.isp': '线路/运营商',
    'node.load': '负载',
    'node.memory': '内存',
    'node.lastSeen': '最后心跳',
    'node.key': '节点密钥',
    'node.keyPrefix': '密钥前缀',
    'node.copyKey': '复制 Key',
    'node.nginx': '查看 Nginx',
    'node.installRemote': '远程节点安装',
    'node.installSame': '同机安装',
    'node.installCommand': '安装命令',
    'node.installHelp': '先创建节点，然后复制对应命令到目标服务器执行。远程节点使用主控公网地址，同机节点使用 127.0.0.1 访问主控。',
    'node.createdTip': '节点已创建，节点密钥已生成。可以在“节点安装”页面复制一键安装命令。',
    'domain.create': '创建域名',
    'domain.list': '域名列表',
    'domain.hostname': '加速域名',
    'domain.tenant': '租户',
    'domain.origin': '源站',
    'domain.cacheTTL': '缓存 TTL 秒',
    'domain.httpsMode': 'HTTPS 模式',
    'domain.wafMode': 'WAF 模式',
    'domain.nodeIds': '节点 ID，多个用英文逗号分隔，留空表示全部节点',
    'domain.verifyTXT': '验证 TXT',
    'plan.create': '创建套餐',
    'plan.list': '套餐列表',
    'plan.bandwidth': '带宽/流量 GB',
    'plan.price': '月费，单位分',
    'plan.features': '功能，多个用英文逗号分隔',
    'key.create': '创建 API Key',
    'key.list': 'API Key 列表',
    'key.scope': '权限范围',
    'key.tenantId': '租户 ID',
    'key.prefix': '前缀',
    'key.createdTip': 'API Key 仅显示一次，请立即复制保存。',
    'docs.edgeAgent': '边缘节点 Agent',
    'docs.remoteTitle': '远程边缘节点一键安装',
    'docs.sameTitle': '主控 + 被控节点同机安装',
    'docs.sameNote': '同机模式不会占用主控的 8080 端口。Nginx 默认监听 80，Agent 通过 127.0.0.1:8080 连接主控。',
    'docs.requirement': '脚本会自动安装 Nginx、Go 编译器、构建 yfcdn-agent、写入 systemd 服务并启动节点心跳。',
    'docs.selectNode': '选择节点',
    'docs.noNode': '请先创建一个节点，创建后会生成节点密钥。',
    'docs.apiExample': 'API 调用示例',
    'status.active': '启用',
    'status.online': '在线',
    'status.offline': '离线',
    'status.paused': '暂停',
    'status.pending': '待验证',
    'status.revoked': '已吊销',
    'status.unknown': '未知'
  },
  'en-US': {
    'app.title': 'YFCDN Control Panel',
    'app.intro': 'Lightweight CDN control plane with multi-tenant operations, edge nodes, domains, API keys and config delivery.',
    'app.footer': 'Original YFCDN lightweight UI template with operator panel and API automation support.',
    'auth.username': 'Username',
    'auth.password': 'Password',
    'auth.login': 'Login',
    'auth.defaultPassword': 'Default password comes from YFCDN_ADMIN_PASSWORD or admin123. Change it before public deployment.',
    'auth.invalid': 'Login failed',
    'auth.expired': 'Session expired. Please sign in again.',
    'top.refresh': 'Refresh',
    'top.logout': 'Logout',
    'top.language': 'Language',
    'common.create': 'Create',
    'common.delete': 'Delete',
    'common.copy': 'Copy',
    'common.copied': 'Copied',
    'common.copyManual': 'Copy manually:',
    'common.noData': 'No data',
    'common.actions': 'Actions',
    'common.status': 'Status',
    'common.created': 'Created',
    'common.active': 'active',
    'common.paused': 'paused',
    'common.offline': 'offline',
    'common.pending': 'pending',
    'common.name': 'Name',
    'common.contact': 'Contact',
    'common.remark': 'Remark',
    'common.all': 'all',
    'common.none': 'none',
    'common.yes': 'yes',
    'common.no': 'no',
    'common.confirmDelete': 'Delete or revoke this item?',
    'common.saved': 'Done',
    'common.loading': 'Loading...',
    'view.dashboard': 'Dashboard',
    'view.tenants': 'Tenants',
    'view.operators': 'Operators',
    'view.nodes': 'Nodes',
    'view.domains': 'Domains',
    'view.plans': 'Plans',
    'view.keys': 'API Keys',
    'view.docs': 'Node Setup',
    'subtitle.dashboard': 'System overview, online nodes and recent events.',
    'subtitle.tenants': 'Manage customers, quotas and tenant status.',
    'subtitle.operators': 'Manage reseller/operator access and API capability.',
    'subtitle.nodes': 'Register edge nodes and inspect heartbeat metrics.',
    'subtitle.domains': 'Bind customer domains to origins and nodes.',
    'subtitle.plans': 'Create commercial bandwidth packages.',
    'subtitle.keys': 'Issue API keys for automation, operators or tenants.',
    'subtitle.docs': 'Install edge agent with one command. All-in-one host supported.',
    'metric.tenants': 'Tenants',
    'metric.onlineNodes': 'Online Nodes',
    'metric.activeDomains': 'Active Domains',
    'metric.traffic': 'Traffic',
    'dashboard.events': 'Recent Events',
    'table.time': 'Time',
    'table.type': 'Type',
    'table.actor': 'Actor',
    'table.message': 'Message',
    'tenant.create': 'Create Tenant',
    'tenant.list': 'Tenants',
    'tenant.quota': 'Traffic quota GB',
    'operator.create': 'Create Operator',
    'operator.list': 'Operators',
    'operator.apiEnabled': 'API enabled',
    'operator.tenantIds': 'Tenant IDs, comma separated',
    'node.create': 'Create Node',
    'node.list': 'Nodes',
    'node.ip': 'IP',
    'node.region': 'Region',
    'node.isp': 'ISP',
    'node.load': 'Load',
    'node.memory': 'Memory',
    'node.lastSeen': 'Last seen',
    'node.key': 'Node key',
    'node.keyPrefix': 'Key prefix',
    'node.copyKey': 'Copy Key',
    'node.nginx': 'Nginx',
    'node.installRemote': 'Remote install',
    'node.installSame': 'Same-host install',
    'node.installCommand': 'Install command',
    'node.installHelp': 'Create a node first, then run the command on the target server. Remote mode uses the public control URL; same-host mode uses 127.0.0.1.',
    'node.createdTip': 'Node created and node key generated. Open Node Setup to copy the one-command installer.',
    'domain.create': 'Create Domain',
    'domain.list': 'Domains',
    'domain.hostname': 'Hostname',
    'domain.tenant': 'Tenant',
    'domain.origin': 'Origin',
    'domain.cacheTTL': 'Cache TTL seconds',
    'domain.httpsMode': 'HTTPS mode',
    'domain.wafMode': 'WAF mode',
    'domain.nodeIds': 'Node IDs, comma separated. Empty means all nodes.',
    'domain.verifyTXT': 'Verify TXT',
    'plan.create': 'Create Plan',
    'plan.list': 'Plans',
    'plan.bandwidth': 'Bandwidth GB',
    'plan.price': 'Monthly price cents',
    'plan.features': 'Features, comma separated',
    'key.create': 'Create API Key',
    'key.list': 'API Keys',
    'key.scope': 'Scope',
    'key.tenantId': 'Tenant ID',
    'key.prefix': 'Prefix',
    'key.createdTip': 'API key is shown only once. Copy and save it now.',
    'docs.edgeAgent': 'Edge Agent',
    'docs.remoteTitle': 'Remote edge one-command install',
    'docs.sameTitle': 'Control + edge on the same server',
    'docs.sameNote': 'Same-host mode does not use the control plane port 8080 for Nginx. Nginx listens on 80 and the agent talks to 127.0.0.1:8080.',
    'docs.requirement': 'The installer installs Nginx, Go, builds yfcdn-agent, writes a systemd service and starts heartbeat/config sync.',
    'docs.selectNode': 'Select node',
    'docs.noNode': 'Create a node first to generate a node key.',
    'docs.apiExample': 'API example',
    'status.active': 'active',
    'status.online': 'online',
    'status.offline': 'offline',
    'status.paused': 'paused',
    'status.pending': 'pending',
    'status.revoked': 'revoked',
    'status.unknown': 'unknown'
  }
};

function t(key) {
  return (i18n[lang] && i18n[lang][key]) || i18n[defaultLang][key] || key;
}

function esc(v) {
  return String(v ?? '').replace(/[&<>'"]/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;',"'":'&#39;','"':'&quot;'}[c]));
}

async function request(path, options = {}) {
  token = localStorage.getItem('yfcdn_token') || token || '';
  const headers = Object.assign({'Content-Type': 'application/json'}, options.headers || {});
  if (token) {
    headers.Authorization = 'Bearer ' + token;
    headers['X-YFCDN-Session'] = token;
  }
  const res = await fetch(apiBase + path, Object.assign({}, options, { headers, credentials: 'include' }));
  const text = await res.text();
  let payload = {};
  try { payload = text ? JSON.parse(text) : {}; } catch (_) { payload = { ok: false, error: text }; }
  if (!res.ok || payload.ok === false) {
    const err = new Error(payload.error || res.statusText);
    err.status = res.status;
    err.payload = payload;
    if (isUnauthorized(err) && path !== '/api/v1/auth/login') clearAuth();
    throw err;
  }
  return payload.data ?? payload;
}

function isUnauthorized(err) {
  return err && (err.status === 401 || /authentication required|session expired/i.test(err.message || ''));
}

function clearAuth() {
  token = '';
  localStorage.removeItem('yfcdn_token');
}

function handleAuthError(err) {
  if (!isUnauthorized(err)) return false;
  clearAuth();
  renderLogin(t('auth.expired'));
  return true;
}

function languageOptions() {
  return `<option value="zh-CN" ${lang === 'zh-CN' ? 'selected' : ''}>简体中文</option><option value="en-US" ${lang === 'en-US' ? 'selected' : ''}>English</option>`;
}

function renderLogin(error = '') {
  document.documentElement.lang = lang;
  document.title = t('app.title');
  app.innerHTML = `
    <div class="login-shell">
      <div class="login-card">
        <div class="login-top">
          <div class="logo"><div class="logo-mark">YF</div><div>YFCDN</div></div>
          <select class="lang-select" data-lang-select aria-label="${esc(t('top.language'))}">${languageOptions()}</select>
        </div>
        <p class="hint">${esc(t('app.intro'))}</p>
        ${error ? `<div class="error">${esc(error)}</div>` : ''}
        <form id="loginForm">
          <div class="field"><label>${esc(t('auth.username'))}</label><input name="username" value="admin" autocomplete="username"></div>
          <div class="field"><label>${esc(t('auth.password'))}</label><input name="password" type="password" value="admin123" autocomplete="current-password"></div>
          <button class="btn" type="submit" style="width:100%;margin-top:8px">${esc(t('auth.login'))}</button>
        </form>
        <p class="footer-note">${esc(t('auth.defaultPassword'))}</p>
      </div>
    </div>
    <div id="toast" class="toast" hidden></div>`;
}

function renderShell(content) {
  document.documentElement.lang = lang;
  document.title = `${titleFor(current)} - YFCDN`;
  app.innerHTML = `
    <div class="shell">
      <aside class="sidebar">
        <div class="logo"><div class="logo-mark">YF</div><div>YFCDN</div></div>
        <nav class="nav">
          ${viewIds.map(id => `<button type="button" data-view="${id}" class="${current === id ? 'active' : ''}">${esc(titleFor(id))}</button>`).join('')}
        </nav>
        <p class="footer-note">${esc(t('app.footer'))}</p>
      </aside>
      <main class="main">
        <div class="topbar">
          <div class="title"><h1>${esc(titleFor(current))}</h1><p>${esc(subtitleFor(current))}</p></div>
          <div class="actions">
            <select class="lang-select" data-lang-select aria-label="${esc(t('top.language'))}">${languageOptions()}</select>
            <button type="button" class="btn secondary" data-action="refresh">${esc(t('top.refresh'))}</button>
            <button type="button" class="btn danger" data-action="logout">${esc(t('top.logout'))}</button>
          </div>
        </div>
        <div id="view">${content}</div>
      </main>
    </div>
    <div id="toast" class="toast" hidden></div>`;
}

function titleFor(view) { return t('view.' + view); }
function subtitleFor(view) { return t('subtitle.' + view); }

async function loadAll() {
  const keep = fallback => err => {
    if (isUnauthorized(err)) throw err;
    return fallback;
  };
  const tasks = [
    request('/api/v1/dashboard').then(v => cache.dashboard = v).catch(keep(null)),
    request('/api/v1/tenants').then(v => cache.tenants = Array.isArray(v) ? v : []).catch(keep([])),
    request('/api/v1/operators').then(v => cache.operators = Array.isArray(v) ? v : []).catch(keep([])),
    request('/api/v1/nodes').then(v => cache.nodes = Array.isArray(v) ? v : []).catch(keep([])),
    request('/api/v1/domains').then(v => cache.domains = Array.isArray(v) ? v : []).catch(keep([])),
    request('/api/v1/plans').then(v => cache.plans = Array.isArray(v) ? v : []).catch(keep([])),
    request('/api/v1/api-keys').then(v => cache.keys = Array.isArray(v) ? v : []).catch(keep([]))
  ];
  await Promise.all(tasks);
}

function renderApp() {
  if (!token) return renderLogin();
  const map = { dashboard: renderDashboard, tenants: renderTenants, operators: renderOperators, nodes: renderNodes, domains: renderDomains, plans: renderPlans, keys: renderKeys, docs: renderDocs };
  let content = '';
  try {
    content = (map[current] || renderDashboard)();
  } catch (err) {
    content = `<div class="panel"><div style="padding:18px" class="error">${esc(err.message || err)}</div></div>`;
  }
  renderShell(content);
}

function metric(label, value) {
  return `<div class="card"><div class="metric">${esc(value)}</div><div class="metric-label">${esc(label)}</div></div>`;
}

function renderDashboard() {
  const d = cache.dashboard || {};
  return `
    <div class="grid">
      ${metric(t('metric.tenants'), d.tenants || 0)}
      ${metric(t('metric.onlineNodes'), `${d.online_nodes || 0}/${d.nodes || 0}`)}
      ${metric(t('metric.activeDomains'), `${d.active_domains || 0}/${d.domains || 0}`)}
      ${metric(t('metric.traffic'), formatBytes((d.rx_bytes || 0) + (d.tx_bytes || 0)))}
    </div>
    <div class="panel"><div class="panel-head"><h2>${esc(t('dashboard.events'))}</h2><span class="badge active">v${esc(d.version || '')}</span></div>${table([t('table.time'), t('table.type'), t('table.actor'), t('table.message')], (d.events || []).map(e => [esc(e.created_at), esc(e.type), esc(e.actor), esc(e.message)]))}</div>`;
}

function table(headers, rows, empty) {
  if (!rows || rows.length === 0) return `<div style="padding:18px" class="hint">${esc(empty || t('common.noData'))}</div>`;
  return `<div class="table-wrap"><table><thead><tr>${headers.map(h => `<th>${esc(h)}</th>`).join('')}</tr></thead><tbody>${rows.map(r => `<tr>${r.map(c => `<td>${c}</td>`).join('')}</tr>`).join('')}</tbody></table></div>`;
}

function statusBadge(s) {
  const key = 'status.' + (s || 'unknown');
  return `<span class="badge ${esc(s || '')}">${esc(t(key) || s || t('status.unknown'))}</span>`;
}

function renderTenants() {
  return `
    <div class="panel"><div class="panel-head"><h2>${esc(t('tenant.create'))}</h2></div>
      <form class="form-grid" data-create="tenant">
        <div class="field"><label>${esc(t('common.name'))}</label><input name="name" required></div>
        <div class="field"><label>${esc(t('common.contact'))}</label><input name="contact"></div>
        <div class="field"><label>${esc(t('common.status'))}</label><select name="status"><option value="active">${esc(t('common.active'))}</option><option value="paused">${esc(t('common.paused'))}</option></select></div>
        <div class="field"><label>${esc(t('tenant.quota'))}</label><input name="traffic_quota_gb" type="number" value="100"></div>
        <div class="field wide"><label>${esc(t('common.remark'))}</label><textarea name="remark"></textarea></div>
        <div><button class="btn" type="submit">${esc(t('common.create'))}</button></div>
      </form>
    </div>
    <div class="panel"><div class="panel-head"><h2>${esc(t('tenant.list'))}</h2></div>${table([t('common.name'), t('common.contact'), t('tenant.quota'), t('common.status'), t('common.created'), t('common.actions')], cache.tenants.map(x => [esc(x.name), esc(x.contact), esc(x.traffic_quota_gb) + ' GB', statusBadge(x.status), esc(x.created_at), delBtn('tenants', x.id)]))}</div>`;
}

function renderOperators() {
  return `
    <div class="panel"><div class="panel-head"><h2>${esc(t('operator.create'))}</h2></div>
      <form class="form-grid" data-create="operator">
        <div class="field"><label>${esc(t('common.name'))}</label><input name="name" required></div>
        <div class="field"><label>${esc(t('common.contact'))}</label><input name="contact"></div>
        <div class="field"><label>${esc(t('common.status'))}</label><select name="status"><option value="active">${esc(t('common.active'))}</option><option value="paused">${esc(t('common.paused'))}</option></select></div>
        <div class="field"><label>${esc(t('operator.apiEnabled'))}</label><select name="api_enabled"><option value="true">${esc(t('common.yes'))}</option><option value="false">${esc(t('common.no'))}</option></select></div>
        <div class="field wide"><label>${esc(t('operator.tenantIds'))}</label><input name="tenant_ids"></div>
        <div><button class="btn" type="submit">${esc(t('common.create'))}</button></div>
      </form>
    </div>
    <div class="panel"><div class="panel-head"><h2>${esc(t('operator.list'))}</h2></div>${table([t('common.name'), t('common.contact'), t('operator.apiEnabled'), t('common.status'), 'Tenants', t('common.actions')], cache.operators.map(x => [esc(x.name), esc(x.contact), esc(x.api_enabled ? t('common.yes') : t('common.no')), statusBadge(x.status), esc((x.tenant_ids || []).join(', ')), delBtn('operators', x.id)]))}</div>`;
}

function renderNodes() {
  return `
    <div class="panel"><div class="panel-head"><h2>${esc(t('node.create'))}</h2></div>
      <form class="form-grid" data-create="node">
        <div class="field"><label>${esc(t('common.name'))}</label><input name="name" required placeholder="edge-01"></div>
        <div class="field"><label>${esc(t('node.ip'))}</label><input name="ip" placeholder="203.0.113.10"></div>
        <div class="field"><label>${esc(t('node.region'))}</label><input name="region" placeholder="Hong Kong"></div>
        <div class="field"><label>${esc(t('node.isp'))}</label><input name="isp" placeholder="BGP"></div>
        <div class="field"><label>${esc(t('common.status'))}</label><select name="status"><option value="active">${esc(t('common.active'))}</option><option value="offline">${esc(t('common.offline'))}</option></select></div>
        <div><button class="btn" type="submit">${esc(t('common.create'))}</button></div>
      </form>
    </div>
    <div class="panel"><div class="panel-head"><h2>${esc(t('node.list'))}</h2><span class="hint">${esc(t('node.installHelp'))}</span></div>${table([t('common.name'), t('node.ip'), t('node.region'), t('node.isp'), t('node.load'), t('node.memory'), t('common.status'), t('node.lastSeen'), t('common.actions')], cache.nodes.map(n => [esc(n.name), esc(n.ip), esc(n.region), esc(n.isp), esc(n.load1), formatBytes(n.memory_used || 0) + ' / ' + formatBytes(n.memory_total || 0), statusBadge(n.status), esc(n.last_seen || '-'), nodeActions(n)]))}</div>`;
}

function nodeActions(n) {
  return `<div class="actions">${delBtn('nodes', n.id)}<button type="button" class="btn secondary small" data-nginx="${esc(n.id)}">${esc(t('node.nginx'))}</button><button type="button" class="btn secondary small" data-copy="${esc(n.node_key || '')}">${esc(t('node.copyKey'))}</button><button type="button" class="btn secondary small" data-copy="${esc(installCommand(n, false))}">${esc(t('node.installRemote'))}</button><button type="button" class="btn secondary small" data-copy="${esc(installCommand(n, true))}">${esc(t('node.installSame'))}</button></div>`;
}

function renderDomains() {
  const tenantOptions = cache.tenants.map(tn => `<option value="${esc(tn.id)}">${esc(tn.name)}</option>`).join('');
  return `
    <div class="panel"><div class="panel-head"><h2>${esc(t('domain.create'))}</h2></div>
      <form class="form-grid" data-create="domain">
        <div class="field"><label>${esc(t('domain.hostname'))}</label><input name="hostname" placeholder="cdn.example.com" required></div>
        <div class="field"><label>${esc(t('domain.tenant'))}</label><select name="tenant_id"><option value="">${esc(t('common.none'))}</option>${tenantOptions}</select></div>
        <div class="field"><label>${esc(t('domain.origin'))}</label><input name="origin" placeholder="https://origin.example.com" required></div>
        <div class="field"><label>${esc(t('domain.cacheTTL'))}</label><input name="cache_ttl" type="number" value="300"></div>
        <div class="field"><label>${esc(t('domain.httpsMode'))}</label><select name="https_mode"><option>off</option><option>edge</option><option>full</option></select></div>
        <div class="field"><label>${esc(t('domain.wafMode'))}</label><select name="waf_mode"><option>basic</option><option>off</option><option>strict</option></select></div>
        <div class="field"><label>${esc(t('common.status'))}</label><select name="status"><option value="active">${esc(t('common.active'))}</option><option value="pending">${esc(t('common.pending'))}</option><option value="paused">${esc(t('common.paused'))}</option></select></div>
        <div class="field wide"><label>${esc(t('domain.nodeIds'))}</label><input name="node_ids" placeholder="nod_xxx,nod_yyy"></div>
        <div><button class="btn" type="submit">${esc(t('common.create'))}</button></div>
      </form>
    </div>
    <div class="panel"><div class="panel-head"><h2>${esc(t('domain.list'))}</h2></div>${table([t('domain.hostname'), t('domain.origin'), 'TTL', 'HTTPS', 'WAF', t('common.status'), 'Nodes', t('domain.verifyTXT'), t('common.actions')], cache.domains.map(d => [esc(d.hostname), esc(d.origin), esc(d.cache_ttl), esc(d.https_mode), esc(d.waf_mode), statusBadge(d.status), esc((d.node_ids || []).join(', ') || t('common.all')), `<code>${esc(d.verification_txt || '-')}</code>`, delBtn('domains', d.id)]))}</div>`;
}

function renderPlans() {
  return `
    <div class="panel"><div class="panel-head"><h2>${esc(t('plan.create'))}</h2></div>
      <form class="form-grid" data-create="plan">
        <div class="field"><label>${esc(t('common.name'))}</label><input name="name" required></div>
        <div class="field"><label>${esc(t('plan.bandwidth'))}</label><input name="bandwidth_gb" type="number" value="100"></div>
        <div class="field"><label>${esc(t('plan.price'))}</label><input name="price_monthly_cents" type="number" value="9900"></div>
        <div class="field"><label>${esc(t('common.status'))}</label><select name="status"><option value="active">${esc(t('common.active'))}</option><option value="paused">${esc(t('common.paused'))}</option></select></div>
        <div class="field wide"><label>${esc(t('plan.features'))}</label><input name="features" value="HTTP cache,Basic WAF,API access"></div>
        <div><button class="btn" type="submit">${esc(t('common.create'))}</button></div>
      </form>
    </div>
    <div class="panel"><div class="panel-head"><h2>${esc(t('plan.list'))}</h2></div>${table([t('common.name'), t('plan.bandwidth'), t('plan.price'), 'Features', t('common.status'), t('common.actions')], cache.plans.map(p => [esc(p.name), esc(p.bandwidth_gb) + ' GB', '$' + ((p.price_monthly_cents || 0) / 100).toFixed(2), esc((p.features || []).join(', ')), statusBadge(p.status), delBtn('plans', p.id)]))}</div>`;
}

function renderKeys() {
  return `
    <div class="panel"><div class="panel-head"><h2>${esc(t('key.create'))}</h2></div>
      <form class="form-grid" data-create="key">
        <div class="field"><label>${esc(t('common.name'))}</label><input name="name" required></div>
        <div class="field"><label>${esc(t('key.scope'))}</label><select name="scope"><option>admin</option><option>operator</option><option>tenant</option></select></div>
        <div class="field"><label>${esc(t('key.tenantId'))}</label><input name="tenant_id"></div>
        <div><button class="btn" type="submit">${esc(t('common.create'))}</button></div>
      </form>
    </div>
    <div class="panel"><div class="panel-head"><h2>${esc(t('key.list'))}</h2></div>${table([t('common.name'), t('key.prefix'), t('key.scope'), t('key.tenantId'), t('common.status'), t('common.created'), t('common.actions')], cache.keys.map(k => [esc(k.name), esc(k.prefix), esc(k.scope), esc(k.tenant_id || '-'), statusBadge(k.status), esc(k.created_at), delBtn('api-keys', k.id)]))}</div>`;
}

function renderDocs() {
  const selectedId = localStorage.getItem('yfcdn_install_node') || (cache.nodes[0] && cache.nodes[0].id) || '';
  const node = cache.nodes.find(n => n.id === selectedId) || cache.nodes[0] || null;
  const nodeOptions = cache.nodes.map(n => `<option value="${esc(n.id)}" ${node && n.id === node.id ? 'selected' : ''}>${esc(n.name)} / ${esc(n.ip || '-')} / ${esc(n.id)}</option>`).join('');
  const remote = installCommand(node, false);
  const same = installCommand(node, true);
  return `
    <div class="panel"><div class="panel-head"><h2>${esc(t('docs.edgeAgent'))}</h2></div>
      <div style="padding:18px">
        <p class="hint">${esc(t('docs.requirement'))}</p>
        ${cache.nodes.length ? `<div class="field"><label>${esc(t('docs.selectNode'))}</label><select data-install-node>${nodeOptions}</select></div>` : `<div class="error">${esc(t('docs.noNode'))}</div>`}
      </div>
    </div>
    <div class="panel"><div class="panel-head"><h2>${esc(t('docs.remoteTitle'))}</h2><button type="button" class="btn secondary small" data-copy="${esc(remote)}">${esc(t('common.copy'))}</button></div><div style="padding:18px"><div class="code">${esc(remote)}</div></div></div>
    <div class="panel"><div class="panel-head"><h2>${esc(t('docs.sameTitle'))}</h2><button type="button" class="btn secondary small" data-copy="${esc(same)}">${esc(t('common.copy'))}</button></div><div style="padding:18px"><p class="hint">${esc(t('docs.sameNote'))}</p><div class="code">${esc(same)}</div></div></div>
    <div class="panel"><div class="panel-head"><h2>${esc(t('docs.apiExample'))}</h2></div><div style="padding:18px"><div class="code">curl -H "Authorization: Bearer YOUR_TOKEN" ${esc(publicControlURL())}/api/v1/domains</div></div></div>`;
}

function delBtn(resource, id) {
  return `<button type="button" class="btn danger small" data-delete="${esc(resource)}:${esc(id)}">${esc(t('common.delete'))}</button>`;
}

async function handleLogin(e) {
  e.preventDefault();
  const form = new FormData(e.target);
  try {
    const data = await request('/api/v1/auth/login', { method: 'POST', body: JSON.stringify(Object.fromEntries(form.entries())) });
    token = data.token;
    localStorage.setItem('yfcdn_token', token);
    await loadAll();
    renderApp();
  } catch (err) {
    renderLogin(err.message || t('auth.invalid'));
  }
}

async function createHandler(e) {
  e.preventDefault();
  const type = e.target.dataset.create;
  const form = new FormData(e.target);
  let body = Object.fromEntries(form.entries());
  try {
    if (type === 'tenant') body.traffic_quota_gb = Number(body.traffic_quota_gb || 0);
    if (type === 'operator') { body.api_enabled = body.api_enabled === 'true'; body.tenant_ids = splitCSV(body.tenant_ids); }
    if (type === 'domain') { body.cache_ttl = Number(body.cache_ttl || 300); body.node_ids = splitCSV(body.node_ids); }
    if (type === 'plan') { body.bandwidth_gb = Number(body.bandwidth_gb || 0); body.price_monthly_cents = Number(body.price_monthly_cents || 0); body.features = splitCSV(body.features); }
    const path = { tenant: '/api/v1/tenants', operator: '/api/v1/operators', node: '/api/v1/nodes', domain: '/api/v1/domains', plan: '/api/v1/plans', key: '/api/v1/api-keys' }[type];
    const data = await request(path, { method: 'POST', body: JSON.stringify(body) });
    e.target.reset();
    await loadAll();
    if (type === 'node') {
      localStorage.setItem('yfcdn_install_node', data.id);
      notify(t('node.createdTip'));
      current = 'docs';
      localStorage.setItem('yfcdn_view', current);
    } else if (type === 'key') {
      await copyText(data.generated_key || '');
      notify(`${t('key.createdTip')} ${data.generated_key || ''}`);
    } else {
      notify(t('common.saved'));
    }
    renderApp();
  } catch (err) {
    if (handleAuthError(err)) return;
    notify(err.message || String(err), 'error');
  }
}

async function deleteHandler(target) {
  const [resource, id] = target.dataset.delete.split(':');
  if (!confirm(t('common.confirmDelete'))) return;
  try {
    await request('/api/v1/' + resource + '/' + id, { method: 'DELETE' });
    await loadAll();
    notify(t('common.saved'));
    renderApp();
  } catch (err) {
    if (handleAuthError(err)) return;
    notify(err.message || String(err), 'error');
  }
}

function publicControlURL() {
  if (window.location && window.location.origin && window.location.origin !== 'null') return window.location.origin.replace(/\/$/, '');
  return 'http://YOUR_CONTROL:8080';
}

function installCommand(node, sameServer) {
  const key = node && node.node_key ? node.node_key : 'node_xxx';
  const control = sameServer ? 'http://127.0.0.1:8080' : publicControlURL();
  const script = sameServer ? 'http://127.0.0.1:8080/install-node.sh' : publicControlURL() + '/install-node.sh';
  return `curl -fsSL ${script} | sudo bash -s -- --control ${control} --node-key ${key}${sameServer ? ' --same-server' : ''}`;
}

function splitCSV(v) { return String(v || '').split(',').map(s => s.trim()).filter(Boolean); }

function formatBytes(n) {
  n = Number(n || 0);
  const units = ['B','KB','MB','GB','TB','PB'];
  let i = 0;
  while (n >= 1024 && i < units.length - 1) { n /= 1024; i++; }
  return n.toFixed(i ? 2 : 0) + ' ' + units[i];
}

async function copyText(text) {
  if (!text) return;
  try {
    await navigator.clipboard.writeText(text);
    notify(t('common.copied'));
  } catch (_) {
    prompt(t('common.copyManual'), text);
  }
}

function notify(message, type) {
  const el = document.getElementById('toast');
  if (!el) return;
  el.textContent = message;
  el.className = 'toast ' + (type || 'info');
  el.hidden = false;
  clearTimeout(notify.timer);
  notify.timer = setTimeout(() => { el.hidden = true; }, 3200);
}

document.addEventListener('submit', async (e) => {
  const form = e.target.closest('form');
  if (!form) return;
  if (form.id === 'loginForm') return handleLogin(e);
  if (form.dataset.create) return createHandler(e);
});

document.addEventListener('click', async (e) => {
  const viewBtn = e.target.closest('[data-view]');
  if (viewBtn) {
    e.preventDefault();
    current = viewBtn.dataset.view;
    localStorage.setItem('yfcdn_view', current);
    renderApp();
    return;
  }

  const action = e.target.closest('[data-action]');
  if (action) {
    e.preventDefault();
    if (action.dataset.action === 'refresh') {
      try {
        await loadAll();
        notify(t('common.saved'));
        renderApp();
      } catch (err) {
        if (!handleAuthError(err)) notify(err.message || String(err), 'error');
      }
    }
    if (action.dataset.action === 'logout') {
      try { await request('/api/v1/auth/logout', { method: 'POST', body: '{}' }); } catch (_) {}
      clearAuth();
      renderLogin();
    }
    return;
  }

  const del = e.target.closest('[data-delete]');
  if (del) {
    e.preventDefault();
    await deleteHandler(del);
    return;
  }

  const copy = e.target.closest('[data-copy]');
  if (copy) {
    e.preventDefault();
    await copyText(copy.dataset.copy);
    return;
  }

  const nginx = e.target.closest('[data-nginx]');
  if (nginx) {
    e.preventDefault();
    try {
      const data = await request('/api/v1/nodes/' + nginx.dataset.nginx + '/nginx');
      await copyText(data.config || '');
      alert(data.config || 'empty config');
    } catch (err) {
      if (handleAuthError(err)) return;
      notify(err.message || String(err), 'error');
    }
  }
});

document.addEventListener('change', (e) => {
  if (e.target.matches('[data-lang-select]')) {
    lang = e.target.value || defaultLang;
    localStorage.setItem('yfcdn_lang', lang);
    renderApp();
  }
  if (e.target.matches('[data-install-node]')) {
    localStorage.setItem('yfcdn_install_node', e.target.value);
    renderApp();
  }
});

(async function boot() {
  document.documentElement.lang = lang;
  if (!token) return renderLogin();
  try {
    await loadAll();
    renderApp();
  } catch (err) {
    clearAuth();
    renderLogin(isUnauthorized(err) ? t('auth.expired') : '');
  }
})();
